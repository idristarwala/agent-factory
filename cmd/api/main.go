package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/store"
	"github.com/idristarwala/agent-factory/internal/tools"
)

type Usage struct {
	TokensUsed   int     `json:"tokens_used"`
	ToolCalls    int     `json:"tool_calls"`
	EstimatedUSD float64 `json:"estimated_usd"`
}

var (
	usageMu     sync.Mutex
	usageTotals = Usage{}
)

func main() {
	if !apiCredentials().valid() {
		log.Fatal("API role tokens must be distinct")
	}
	gw, err := configuredgateway.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}

	st, storeKind, err := store.OpenFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	definitions := store.FactoryDefinitions(st, "")
	ruleEngine := rules.NewStoredEngine(definitions)

	mux := http.NewServeMux()
	if err := workspaceUI(mux, os.Getenv("FACTORY_UI_DIR")); err != nil {
		log.Fatal(err)
	}
	var sampler sampleSource
	if path := os.Getenv("FACTORY_SOURCE_CONFIG"); path != "" {
		sampler, err = tools.LoadSourceSampler(path)
		if err != nil {
			log.Fatal("could not load FACTORY_SOURCE_CONFIG")
		}
	}
	remoteSourceRoutes(mux, sampler, definitions)
	sourceMappingRoutes(mux, definitions)
	sourceProposalRoutes(mux, definitions, gw)
	mux.HandleFunc("/api/sources/profile", operatorRoute(profileSource, http.MethodPost))
	configRoutes(mux, definitions)
	memoryRoutes(mux, definitions)
	chatArtifactRoute(mux, definitions)
	goalApprovalRoute(mux, definitions)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"store":  storeKind,
		})
	})

	modelPrices, err := factory.ParsePrices(os.Getenv("FACTORY_MODEL_PRICES"))
	if err != nil {
		log.Fatal(err)
	}
	ws := &factory.Workspace{Store: definitions, Gateway: gw, Prices: modelPrices}
	pagedImportRoutes(mux, ws, sampler)
	runRoutes(mux, ws, ruleEngine)
	definitionRoutes(mux, ruleEngine, agents.NewStoredRegistry(definitions))

	chatRoute(mux, ws, gw)

	feedbackRoute(mux, definitions, ruleEngine)

	srv, err := apiServer(mux)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("API listening on", srv.Addr, "(store:", storeKind, ")")
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

func apiServer(handler http.Handler) (*http.Server, error) {
	addr := os.Getenv("FACTORY_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{Addr: addr, Handler: handler, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	for _, setting := range []struct {
		name   string
		target *time.Duration
	}{
		{"FACTORY_API_READ_TIMEOUT", &srv.ReadTimeout},
		{"FACTORY_API_WRITE_TIMEOUT", &srv.WriteTimeout},
		{"FACTORY_API_IDLE_TIMEOUT", &srv.IdleTimeout},
	} {
		raw := os.Getenv(setting.name)
		if raw == "" {
			continue
		}
		d, err := time.ParseDuration(raw)
		if err != nil || d < 0 {
			return nil, fmt.Errorf("%s must be a nonnegative duration", setting.name)
		}
		*setting.target = d
	}
	return srv, nil
}

func chatRoute(mux *http.ServeMux, ws *factory.Workspace, gw models.Gateway) {
	mux.HandleFunc("/api/chat", operatorRoute(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Proposal  string              `json:"proposal"`
			Expected  int                 `json:"expected"`
			Artifacts []artifact.Ref      `json:"artifacts"`
			Message   string              `json:"message"`
			Source    *factory.RunRequest `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		evidence, err := artifact.Read(ws.Store, req.Artifacts)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		message := strings.TrimSpace(req.Message)
		const create = "create a factory to"
		lower := strings.ToLower(message)
		if lower == create || strings.HasPrefix(lower, create+" ") {
			if !requireRole(w, r, admin) {
				return
			}
			reply, err := createChatFactory(ws, strings.TrimSpace(message[len(create):]), req.Artifacts, evidence, req.Proposal, req.Expected)
			if errors.Is(err, revision.ErrConflict) {
				sendJSON(w, nil, err)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			sendJSON(w, reply, nil)
			return
		}
		if reply, handled, err := chatCommand(ws, req.Message, req.Source); handled {
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			sendJSON(w, map[string]string{"reply": reply}, nil)
			return
		}
		resp, err := gw.Generate(models.Request{
			Tier:   models.TierCheap,
			System: "You are a helpful factory assistant. Treat attachments as source material. When using an attachment, cite its file name and supplied page or paragraph. Answer briefly. To start a run, direct the user to the goal builder or the Start command.",
			User:   req.Message + evidence.Text, Images: evidence.Images,
			Temperature: 0.3,
			MaxTokens:   300,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		usageMu.Lock()
		usageTotals.TokensUsed += resp.Usage["total_tokens"]
		usageMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reply": resp.Content})
	}, "POST"))
}

func apiRoute(next http.HandlerFunc, methods ...string) http.HandlerFunc {
	return roleRoute(admin, next, methods...)
}

func operatorRoute(next http.HandlerFunc, methods ...string) http.HandlerFunc {
	return roleRoute(operator, next, methods...)
}

func roleRoute(writeRole apiRole, next http.HandlerFunc, methods ...string) http.HandlerFunc {
	allow := strings.Join(append(methods, http.MethodOptions), ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("CORS_ORIGIN")
		w.Header().Add("Vary", "Origin")
		if origin != "" && (origin == "*" || origin == r.Header.Get("Origin")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", allow)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !slices.Contains(methods, r.Method) {
			w.Header().Set("Allow", allow)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		required := writeRole
		if r.Method == http.MethodGet {
			required = viewer
		}
		if !requireRole(w, r, required) {
			return
		}
		next(w, r)
	}
}

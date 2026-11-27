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

	st, err := store.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	definitions := store.FactoryDefinitions(st, "")
	ruleEngine := rules.NewStoredEngine(definitions)

	mux := http.NewServeMux()
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
		storeKind := storeKindLabel()
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

	addr := os.Getenv("FACTORY_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Println("API listening on", addr, "(store:", storeKindLabel(), ")")
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
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

func storeKindLabel() string {
	if os.Getenv("FACTORY_STORE_MODE") == "memory" {
		return "memory"
	}
	if os.Getenv("DATABASE_URL") != "" {
		return "postgres"
	}
	return "sqlite"
}

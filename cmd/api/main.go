package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/store"
)

type Goal struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Decision struct {
	ID       string `json:"id"`
	GoalID   string `json:"goal_id"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type Usage struct {
	TokensUsed   int     `json:"tokens_used"`
	ToolCalls    int     `json:"tool_calls"`
	EstimatedUSD float64 `json:"estimated_usd"`
}

var (
	goalsMu     sync.Mutex
	goals       = map[string]*Goal{}
	goalSeq     int
	decisions   = []Decision{}
	usageMu     sync.Mutex
	usageTotals = Usage{}
)

func main() {
	gw, err := models.NewGatewayFromEnv()
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
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		storeKind := "sqlite"
		if os.Getenv("DATABASE_URL") != "" {
			storeKind = "postgres"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"store":  storeKind,
		})
	})

	mux.HandleFunc("/api/goals", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			goalsMu.Lock()
			goalSeq++
			id := fmt.Sprintf("goal-%d", goalSeq)
			g := &Goal{ID: id, Description: body.Description, Status: "created", CreatedAt: time.Now()}
			goals[id] = g
			goalsMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(g)
			return
		}
		goalsMu.Lock()
		list := make([]*Goal, 0, len(goals))
		for _, g := range goals {
			list = append(list, g)
		}
		goalsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	}, "GET", "POST"))

	mux.HandleFunc("/api/decisions", apiRoute(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(decisions)
	}, "GET"))

	mux.HandleFunc("/api/usage", apiRoute(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		usageMu.Lock()
		snapshot := usageTotals
		usageMu.Unlock()
		json.NewEncoder(w).Encode(snapshot)
	}, "GET"))

	definitionRoutes(mux, ruleEngine, agents.NewStoredRegistry(definitions))

	mux.HandleFunc("/api/graph", apiRoute(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		// Placeholder graph snapshot for the UI explorer
		json.NewEncoder(w).Encode(map[string]any{
			"nodes": []map[string]string{},
			"edges": []map[string]string{},
		})
	}, "GET"))

	mux.HandleFunc("/api/chat", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := gw.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You are a helpful factory assistant. Answer briefly. If the user gives a goal, confirm you will start a factory run for it.",
			User:        req.Message,
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

	mux.HandleFunc("/api/feedback", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Note   string `json:"note"`
			Entity string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Note) == "" {
			http.Error(w, "note is required", 400)
			return
		}
		rule, err := ruleEngine.Save(rules.VersionedRule{Description: body.Note, Condition: body.Entity, Action: "constrain future proposals", Source: "feedback", CreatedBy: "customer", Active: true}, 0)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "rule_created",
			"rule_id": rule.ID,
			"version": rule.Version,
			"source":  "feedback",
		})
	}, "POST"))

	addr := ":8080"
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

func apiRoute(next http.HandlerFunc, methods ...string) http.HandlerFunc {
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
		token := os.Getenv("API_TOKEN")
		if token == "" {
			next(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func storeKindLabel() string {
	if os.Getenv("DATABASE_URL") != "" {
		return "postgres"
	}
	return "sqlite"
}

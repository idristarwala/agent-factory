package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"

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
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	Approved  bool      `json:"approved"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type Usage struct {
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	ToolCalls int     `json:"tool_calls"`
	CostUSD   float64 `json:"cost_usd"`
}

var (
	goalsMu     sync.Mutex
	goals       = map[string]*Goal{}
	goalSeq     int
	decisionsMu sync.Mutex
	decisions   = map[string]*Decision{}
	decisionSeq int
	usageMu     sync.Mutex
	usage       = Usage{}
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

	ruleEngine := rules.NewEngine()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		storeKind := "memory"
		if os.Getenv("DATABASE_URL") != "" {
			storeKind = "postgres"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"store":  storeKind,
		})
	})

	mux.HandleFunc("/api/goals", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
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
		// GET
		goalsMu.Lock()
		list := make([]*Goal, 0, len(goals))
		for _, g := range goals {
			list = append(list, g)
		}
		goalsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	}))

	mux.HandleFunc("/api/decisions", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Content-Type", "application/json")
		decisionsMu.Lock()
		list := make([]*Decision, 0, len(decisions))
		for _, d := range decisions {
			list = append(list, d)
		}
		decisionsMu.Unlock()
		json.NewEncoder(w).Encode(list)
	}))

	mux.HandleFunc("/api/rules", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ruleEngine.Active())
	}))

	mux.HandleFunc("/api/graph", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"nodes": []any{}, "edges": []any{}})
	}))

	mux.HandleFunc("/api/usage", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Content-Type", "application/json")
		usageMu.Lock()
		defer usageMu.Unlock()
		json.NewEncoder(w).Encode(usage)
	}))

	mux.HandleFunc("/api/chat", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := gw.Generate(models.Request{
			Tier:        models.TierCheap,
			System:      "You are a helpful catalog assistant. Answer briefly. If the user gives a goal, confirm you will start a factory run for it.",
			User:        req.Message,
			Temperature: 0.3,
			MaxTokens:   300,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reply": resp.Content})
	}))

	mux.HandleFunc("/api/feedback", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Note   string `json:"note"`
			Entity string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rule := ruleEngine.FromFeedback(body.Note, body.Entity, "customer")
		if rule.ID != "" {
			_ = st.SaveRuleVersion(rule.ID, rule.Version, []byte(body.Note))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "rule_created",
			"rule_id": rule.ID,
			"version": rule.Version,
			"source":  "feedback",
		})
	}))

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

func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	return "memory"
}

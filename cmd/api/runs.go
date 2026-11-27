package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func runResponse(w http.ResponseWriter, v any, err error) {
	if errors.Is(err, factory.ErrNotFound) {
		http.Error(w, "run or decision not found", 404)
		return
	}
	sendJSON(w, v, err)
}
func startRun(ws *factory.Workspace, req factory.RunRequest) (factory.Run, error) {
	run, err := ws.Start(req)
	if err == nil && run.Status == "queued" {
		go func() {
			if err := ws.Execute(run.ID); err != nil && !errors.Is(err, revision.ErrConflict) {
				log.Printf("run %s: %v", run.ID, err)
			}
		}()
	}
	return run, err
}
func runRoutes(mux *http.ServeMux, ws *factory.Workspace, engine *rules.Engine) {
	mux.HandleFunc("/api/goals", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			list, err := ws.List()
			runResponse(w, list, err)
			return
		}
		var req factory.RunRequest
		if !readEdit(w, r, &req) {
			return
		}
		run, err := startRun(ws, req)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		runResponse(w, run, nil)
	}, "GET", "POST"))
	mux.HandleFunc("/api/goals/{id}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var run factory.Run
		var err error
		if r.Method == "DELETE" {
			run, err = ws.Cancel(r.PathValue("id"))
		} else {
			run, err = ws.Get(r.PathValue("id"))
		}
		runResponse(w, run, err)
	}, "GET", "DELETE"))
	mux.HandleFunc("/api/decisions", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			list, err := ws.List()
			if err != nil {
				runResponse(w, nil, err)
				return
			}
			out := []factory.Review{}
			for _, run := range list {
				if id := r.URL.Query().Get("run"); id == "" || run.ID == id {
					out = append(out, run.Decisions...)
				}
			}
			runResponse(w, out, nil)
			return
		}
		var edit struct {
			Run, ID, Status, Feedback string
			Expected                  int
		}
		if !readEdit(w, r, &edit) {
			return
		}
		if edit.Status != "approved" && edit.Status != "rejected" {
			http.Error(w, "invalid status", 400)
			return
		}
		run, err := ws.Decide(edit.Run, edit.ID, edit.Status, edit.Feedback, edit.Expected)
		if err == nil && strings.TrimSpace(edit.Feedback) != "" {
			// The durable review is the source of feedback. A stable key allows retry after a partial failure.
			err = engine.Install(rules.VersionedRule{ID: "feedback/" + edit.ID, Description: edit.Feedback, Condition: edit.ID, Source: "feedback", CreatedBy: "human", Active: !run.Settings.ReplayGate})
		}
		runResponse(w, run, err)
	}, "GET", "POST"))
	mux.HandleFunc("/api/graph", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		list, err := ws.List()
		if err != nil {
			runResponse(w, nil, err)
			return
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if offset < 0 || limit < 0 {
			http.Error(w, "invalid pagination", 400)
			return
		}
		if limit == 0 || limit > 200 {
			limit = 100
		}
		nodes := []any{}
		edges := []any{}
		total := 0
		for _, run := range list {
			if id := r.URL.Query().Get("run"); id != "" && run.ID != id {
				continue
			}
			for _, e := range run.Records {
				total++
				if total <= offset || len(nodes) >= limit {
					continue
				}
				links := []string{}
				for _, d := range run.Decisions {
					if d.RecordID == e.ID {
						links = append(links, d.ID)
					}
				}
				nodes = append(nodes, map[string]any{"id": e.ID, "type": e.Type, "attributes": e.Attributes, "run": run.ID, "decisions": links})
				for _, rel := range e.Relations {
					edges = append(edges, map[string]any{"from": e.ID, "to": rel.Target, "type": rel.Type, "run": run.ID})
				}
			}
		}
		runResponse(w, map[string]any{"nodes": nodes, "edges": edges, "total": total}, nil)
	}, "GET"))
	mux.HandleFunc("/api/usage", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		list, err := ws.List()
		if err != nil {
			runResponse(w, nil, err)
			return
		}
		total := 0
		cost := 0.0
		calls := 0
		unknown := 0
		for _, g := range list {
			if id := r.URL.Query().Get("run"); id != "" && id != g.ID {
				continue
			}
			for _, c := range g.Calls {
				total += c.Total
				cost += c.Cost
				calls++
				if !c.Priced {
					unknown++
				}
			}
		}
		usageMu.Lock()
		chat := usageTotals
		usageMu.Unlock()
		if r.URL.Query().Get("run") == "" {
			total += chat.TokensUsed
			cost += chat.EstimatedUSD
		}
		runResponse(w, map[string]any{"tokens_used": total, "estimated_usd": cost, "tool_calls": 0, "model_calls": calls, "unpriced_calls": unknown}, nil)
	}, "GET"))
}
func chatCommand(ws *factory.Workspace, message string, source *factory.RunRequest) (string, bool, error) {
	words := strings.Fields(message)
	if len(words) == 0 {
		return "", false, nil
	}
	switch strings.ToLower(words[0]) {
	case "status":
		if len(words) != 2 {
			return "Use status followed by a run ID.", true, nil
		}
		r, err := ws.Get(words[1])
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("%s is %s with %d reviewed records.", r.ID, r.Status, len(r.Decisions)), true, nil
	case "start":
		if source == nil || len(source.Records) == 0 {
			return "Choose a factory and source records in the goal builder, then send Start with your goal.", true, nil
		}
		req := *source
		req.Description = strings.TrimSpace(strings.Join(words[1:], " "))
		r, err := startRun(ws, req)
		if err != nil {
			return "", true, err
		}
		return "Started run " + r.ID + " using factory revision " + strconv.Itoa(r.Plan.Revision) + ".", true, nil
	}
	return "", false, nil
}
func prices(raw string) (map[string]factory.Price, error) {
	p := map[string]factory.Price{}
	if raw == "" {
		return p, nil
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	for _, v := range p {
		if v.Input < 0 || v.Output < 0 {
			return nil, fmt.Errorf("negative model price")
		}
	}
	return p, nil
}

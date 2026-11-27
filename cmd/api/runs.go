package main

import (
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
func requestedRuns(ws *factory.Workspace, id string) ([]factory.Run, error) {
	if id == "" {
		return ws.List()
	}
	run, err := ws.Get(id)
	if err != nil {
		return nil, err
	}
	return []factory.Run{run}, nil
}

func runRoutes(mux *http.ServeMux, ws *factory.Workspace, engine *rules.Engine) {
	mux.HandleFunc("/api/goals", operatorRoute(func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/goals/{id}", operatorRoute(func(w http.ResponseWriter, r *http.Request) {
		var run factory.Run
		var err error
		if r.Method == "DELETE" {
			run, err = ws.Cancel(r.PathValue("id"))
		} else {
			run, err = ws.Get(r.PathValue("id"))
		}
		runResponse(w, run, err)
	}, "GET", "DELETE"))
	mux.HandleFunc("/api/decisions", operatorRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			list, err := requestedRuns(ws, r.URL.Query().Get("run"))
			if err != nil {
				runResponse(w, nil, err)
				return
			}
			out := []factory.Review{}
			for _, run := range list {
				out = append(out, run.Decisions...)
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
	mux.HandleFunc("/api/decisions/{id...}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		runs, err := requestedRuns(ws, r.URL.Query().Get("run"))
		if err != nil {
			runResponse(w, nil, err)
			return
		}
		for _, run := range runs {
			for _, decision := range run.Decisions {
				if decision.ID == r.PathValue("id") {
					runResponse(w, decision, nil)
					return
				}
			}
		}
		runResponse(w, nil, factory.ErrNotFound)
	}, "GET"))
	mux.HandleFunc("/api/graph", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		list, err := requestedRuns(ws, r.URL.Query().Get("run"))
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
		list, err := requestedRuns(ws, r.URL.Query().Get("run"))
		if err != nil {
			runResponse(w, nil, err)
			return
		}
		usage := runUsage(list, r.URL.Query().Get("run"))
		if r.URL.Query().Get("run") == "" {
			usageMu.Lock()
			usage.ChatTokens = usageTotals.TokensUsed
			usageMu.Unlock()
		}
		usage.Tokens = usage.RunTokens + usage.ChatTokens
		runResponse(w, usage, nil)
	}, "GET"))
}

// chatCommand resolves run operations against the workspace so replies reflect durable state.
func chatCommand(ws *factory.Workspace, message string, source *factory.RunRequest) (string, bool, error) {
	text := strings.TrimSpace(message)
	for _, prefix := range []string{"please ", "can you ", "could you "} {
		if strings.HasPrefix(strings.ToLower(text), prefix) {
			text = strings.TrimSpace(text[len(prefix):])
		}
	}
	text = strings.TrimRight(text, "?!")
	lower := strings.ToLower(text)
	for _, prefix := range []string{"what is the status of ", "what's the status of ", "check the status of ", "check status of ", "status of ", "status "} {
		if strings.HasPrefix(lower, prefix) {
			id := strings.TrimSpace(text[len(prefix):])
			if strings.HasPrefix(strings.ToLower(id), "run ") {
				id = strings.TrimSpace(id[4:])
			}
			switch strings.ToLower(id) {
			case "latest", "latest run", "the latest run", "my latest run":
				runs, err := ws.List()
				if err != nil {
					return "", true, err
				}
				if len(runs) == 0 {
					return "There are no runs in this workspace yet.", true, nil
				}
				id = runs[0].ID
			}
			r, err := ws.Get(id)
			if err != nil {
				return "", true, err
			}
			return fmt.Sprintf("%s is %s. Decisions: %d. Source records: %d.", r.ID, r.Status, len(r.Decisions), len(r.Records)), true, nil
		}
	}
	if lower == "status" || lower == "check status" || lower == "what is the status" {
		return "Which run? Send its ID after Status, or ask for the status of the latest run.", true, nil
	}
	for _, prefix := range []string{"start a new factory to", "start a factory to", "start a run to", "start a run", "start"} {
		if lower != prefix && !strings.HasPrefix(lower, prefix+" ") {
			continue
		}
		goal := strings.TrimSpace(text[len(prefix):])
		if goal == "" {
			return "What should this run accomplish?", true, nil
		}
		if source == nil || (len(source.Records) == 0 && source.Source == nil && source.Import == nil) || source.Factory == "" {
			return "Choose a factory and source records in the goal builder, then send Start with your goal.", true, nil
		}
		req := *source
		req.Description = goal
		r, err := startRun(ws, req)
		if err != nil {
			return "", true, err
		}
		return "Started run " + r.ID + " using factory revision " + strconv.Itoa(r.Plan.Revision) + ".", true, nil
	}
	return "", false, nil
}

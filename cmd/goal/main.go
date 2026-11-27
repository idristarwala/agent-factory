package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/store"
)

func main() {
	name := flag.String("factory", os.Getenv("FACTORY_DEFINITION"), "factory definition ID")
	version := flag.Int("revision", 0, "definition revision (zero selects latest)")
	path := flag.String("records", os.Getenv("FACTORY_RECORDS"), "JSON source file containing an array of entities")
	flag.Parse()
	goal := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if goal == "" || *name == "" || *path == "" {
		log.Fatal("usage: agent-factory -factory ID -records records.json [-revision N] GOAL")
	}
	st, err := store.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	plan, err := factory.Prepare(store.FactoryDefinitions(st, ""), *name, *version)
	if err != nil {
		log.Fatal(err)
	}
	raw, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}
	var records []schema.Entity
	if err = json.Unmarshal(raw, &records); err != nil {
		log.Fatal(err)
	}
	if len(records) == 0 {
		log.Fatal("source contains no records")
	}
	seen := map[string]bool{}
	for _, e := range records {
		if e.Type != plan.Spec.RecordType || e.ID == "" || seen[e.ID] {
			log.Fatalf("invalid or duplicate source record %q", e.ID)
		}
		seen[e.ID] = true
	}
	gw, err := configuredgateway.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Factory %s revision %d, steward r%d, worker r%d, judge r%d\n", plan.Factory, plan.Revision, plan.Steward.Revision, plan.Worker.Revision, plan.Judge.Revision)
	for _, e := range records {
		m, d, err := plan.Evaluate(gw, goal, e)
		if err != nil {
			log.Fatal(err)
		}
		if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"entity_id": e.ID, "factory_revision": plan.Revision, "proposal": m, "decision": d}); err != nil {
			log.Fatal(err)
		}
	}
}

package factory

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/idristarwala/agent-factory/internal/revision"
)

// ObservationRef identifies the immutable human-reviewed run used for promotion.
type ObservationRef struct {
	Run     string `json:"run"`
	Version int    `json:"version"`
}

func verifyPackObservation(st revision.Store, plan Plan) error {
	ref := plan.Spec.PackPromotion
	if ref.Run == "" || ref.Version < 1 {
		return errors.New("pack_promotion requires an observation run and positive revision")
	}
	history, err := st.Read(revision.Run, ref.Run)
	if err != nil {
		return err
	}
	if ref.Version > len(history) {
		return errors.New("unknown pack observation revision")
	}
	var observed Run
	if err := decodeRun(history[ref.Version-1].Data, &observed); err != nil {
		return err
	}
	if observed.Status != "completed" || !observed.Settings.Observe || observed.Plan.Factory != plan.Factory || observed.Plan.ApprovedPack == nil || observed.Plan.ApprovedPack.PackRef != plan.ApprovedPack.PackRef {
		return errors.New("promotion requires a completed observation of the same factory and pack revision")
	}
	want := plan.Spec
	want.PackPromotion = nil
	previous := observed.Plan.Spec
	previous.PackPromotion = nil
	// Compare the persisted wire form so omitted optional slices match empty ones.
	previousJSON, err := json.Marshal(previous)
	if err != nil {
		return err
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		return err
	}
	if !bytes.Equal(previousJSON, wantJSON) {
		return errors.New("factory configuration differs from the reviewed observation")
	}
	for _, d := range observed.Decisions {
		if d.Status == "approved" {
			return nil
		}
	}
	return errors.New("observation requires a human-approved decision")
}

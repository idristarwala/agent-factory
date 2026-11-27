package factory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/revision"
)

type sampleReceipt struct {
	Factory string `json:"factory"`
	Feed    string `json:"feed"`
	Digest  string `json:"sha256"`
}

func sampleDigest(payload any) (string, error) {
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(normalized)), nil
}

// SaveSampleReceipt attests to the payload fetched by the configured feed.
// Only its fingerprint and source identifiers enter the revision store.
func SaveSampleReceipt(s revision.Store, factory, feed string, payload any) (string, error) {
	digest, err := sampleDigest(payload)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(sampleReceipt{factory, feed, digest})
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	_, err = revision.SaveDocument(s, "source-receipt/"+id, 0, "sampler", data)
	return id, err
}

func (w *Workspace) verifiedFeed(factory, id string, payload any) (string, error) {
	docs, err := revision.Documents(w.Store, "source-receipt/"+id)
	if err != nil {
		return "", err
	}
	if len(docs) != 1 {
		return "", fmt.Errorf("unrecognized sample receipt")
	}
	var receipt sampleReceipt
	if err := json.Unmarshal(docs[0].Data, &receipt); err != nil {
		return "", err
	}
	digest, err := sampleDigest(payload)
	if err != nil {
		return "", err
	}
	if receipt.Factory != factory || digest != receipt.Digest {
		return "", fmt.Errorf("sample receipt does not match this factory and payload")
	}
	return receipt.Feed, nil
}

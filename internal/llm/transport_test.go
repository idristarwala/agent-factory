package llm

import (
	"testing"
)

func TestNewHTTPClientNoCA(t *testing.T) {
	c, err := NewHTTPClient("", "none")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("expected client")
	}
}

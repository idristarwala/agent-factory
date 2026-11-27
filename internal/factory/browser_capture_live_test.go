//go:build browser_live

package factory

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/configuredgateway"
)

func TestLiveBrowserEvidence(t *testing.T) {
	gateway, err := configuredgateway.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	exerciseBrowserRun(t, gateway)
}

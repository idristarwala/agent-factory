package mapping

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCompositeSuggestions(t *testing.T) {
	rows := []any{}
	for _, zone := range []string{"west", "east"} {
		for _, n := range []json.Number{"9007199254740992", "9007199254740993"} {
			rows = append(rows, map[string]any{"zone": zone, "port": map[string]any{"n": n}, "empty": "   "})
		}
	}
	profiles, err := ProfileSample(rows)
	if err != nil || !reflect.DeepEqual(profiles[0].CompositeKeys, [][]string{{"/port/n", "/zone"}}) {
		t.Fatal(profiles, err)
	}
	rows = append(rows, rows[0])
	profiles, err = ProfileSample(rows)
	if err != nil || len(profiles[0].CompositeKeys) != 0 {
		t.Fatal(profiles, err)
	}
}

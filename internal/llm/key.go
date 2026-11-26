package llm

import (
	"os"
	"strings"
)

// LoadKey reads a key from a file or returns an inline value.
// Supports the key_file requirement from issue #11.
func LoadKey(keyFile, inline string) (string, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return inline, nil
}

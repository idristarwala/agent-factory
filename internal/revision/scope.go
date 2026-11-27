package revision

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// A named factory can share a database without observing another factory's rules.
// The default factory retains the keys used by versions predating scoped history.
type Scope struct {
	Store Store
	Name  string
}

const namespace = "@factory:"

func (s Scope) prefix() string {
	if s.Name == "" {
		return ""
	}
	return namespace + base64.RawURLEncoding.EncodeToString([]byte(s.Name)) + "/"
}
func (s Scope) Read(k Kind, id string) ([]Snapshot, error) {
	if strings.HasPrefix(id, namespace) {
		return nil, fmt.Errorf("reserved definition ID")
	}
	key := ""
	prefix := s.prefix()
	if id != "" {
		key = prefix + id
	}
	rows, err := s.Store.Read(k, key)
	if err != nil {
		return nil, err
	}
	out := make([]Snapshot, 0, len(rows))
	for _, v := range rows {
		if prefix == "" && strings.HasPrefix(v.ID, namespace) {
			continue
		}
		local, ok := strings.CutPrefix(v.ID, prefix)
		if ok {
			v.ID = local
			out = append(out, v)
		}
	}
	return out, nil
}
func (s Scope) Insert(k Kind, v Snapshot) error {
	if v.ID == "" || strings.HasPrefix(v.ID, namespace) {
		return fmt.Errorf("invalid definition ID")
	}
	v.ID = s.prefix() + v.ID
	return s.Store.Insert(k, v)
}

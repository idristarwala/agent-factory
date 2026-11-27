package store

import (
	"database/sql"
	"encoding/json"
	"github.com/idristarwala/agent-factory/internal/graph"
	"strings"
)

// nodeRow is the shared storage representation for JSON graph properties.
type nodeRow struct {
	id         graph.NodeID
	typ, props string
}

func encodeNode(n graph.Node) (nodeRow, error) {
	b, err := json.Marshal(n.Props)
	return nodeRow{n.ID, n.Type, string(b)}, err
}
func (row nodeRow) node() (*graph.Node, error) {
	n := &graph.Node{ID: row.id, Type: row.typ}
	if row.props != "" {
		dec := json.NewDecoder(strings.NewReader(row.props))
		dec.UseNumber()
		if err := dec.Decode(&n.Props); err != nil {
			return nil, err
		}
	}
	return n, nil
}
func readNode(db *sql.DB, id graph.NodeID) (*graph.Node, error) {
	row := nodeRow{id: id}
	if err := db.QueryRow("SELECT type, COALESCE(props,'') FROM nodes WHERE id=$1", id).Scan(&row.typ, &row.props); err != nil {
		return nil, err
	}
	return row.node()
}

type nodeWriter interface {
	Exec(string, ...any) (sql.Result, error)
}

func writeNode(db nodeWriter, n graph.Node) error {
	row, err := encodeNode(n)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO nodes (id,type,props) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET type=excluded.type, props=excluded.props", row.id, row.typ, row.props)
	return err
}
func writeMutation(db *sql.DB, m *graph.Mutation) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, nodes := range [][]graph.Node{m.Updates, m.Adds} {
		for _, n := range nodes {
			if err = writeNode(tx, n); err != nil {
				return err
			}
		}
	}
	for _, id := range m.Deletes {
		if _, err = tx.Exec("DELETE FROM nodes WHERE id=$1", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

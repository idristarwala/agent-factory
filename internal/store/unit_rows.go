package store

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

type unitRow struct {
	id, goal, description string
	status                unitofwork.Status
	payload, proposal     string
}

func proposalJSON(m *graph.Mutation) (string, error) {
	data, err := json.Marshal(m)
	return string(data), err
}
func encodeUnit(u *unitofwork.UnitOfWork) (unitRow, error) {
	copy := *u
	copy.Proposal = nil
	data, err := json.Marshal(copy)
	if err != nil {
		return unitRow{}, err
	}
	proposal, err := proposalJSON(u.Proposal)
	return unitRow{id: u.ID, goal: u.GoalID, description: u.Description, status: u.Status, payload: string(data), proposal: proposal}, err
}
func (row unitRow) unit() (*unitofwork.UnitOfWork, error) {
	var u unitofwork.UnitOfWork
	if row.payload != "" {
		d := json.NewDecoder(strings.NewReader(row.payload))
		d.UseNumber()
		if err := d.Decode(&u); err != nil {
			return nil, err
		}
	}
	u.ID, u.GoalID, u.Description, u.Status = row.id, row.goal, row.description, row.status
	if row.proposal != "" {
		d := json.NewDecoder(strings.NewReader(row.proposal))
		d.UseNumber()
		if err := d.Decode(&u.Proposal); err != nil {
			return nil, err
		}
	}
	return &u, nil
}
func createUnit(db *sql.DB, u *unitofwork.UnitOfWork) error {
	row, err := encodeUnit(u)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO units(id,goal_id,description,status,proposal) VALUES ($1,$2,$3,$4,$5)", row.id, row.goal, row.description, row.status, row.proposal); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO unit_payloads(id,data) VALUES ($1,$2)", row.id, row.payload); err != nil {
		return err
	}
	return tx.Commit()
}
func readUnit(db *sql.DB, id string) (*unitofwork.UnitOfWork, error) {
	row := unitRow{id: id}
	err := db.QueryRow("SELECT goal_id,description,status,COALESCE(proposal,''),COALESCE(p.data,'') FROM units u LEFT JOIN unit_payloads p ON p.id=u.id WHERE u.id=$1", id).Scan(&row.goal, &row.description, &row.status, &row.proposal, &row.payload)
	if err != nil {
		return nil, err
	}
	return row.unit()
}
func unitUpdate(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
func attachProposal(db *sql.DB, id string, m *graph.Mutation) error {
	data, err := proposalJSON(m)
	if err != nil {
		return err
	}
	return unitUpdate(db.Exec("UPDATE units SET proposal=$1 WHERE id=$2", data, id))
}

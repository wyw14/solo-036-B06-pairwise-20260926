package service

import (
	"fmt"
	"github.com/wyw14/solo-036/internal/journal"
	"github.com/wyw14/solo-036/internal/model"
	"github.com/wyw14/solo-036/internal/run"
)

func (r *Runtime) CreateOperation(station string) (model.Run, error) {
	// Prepare the run and its creation event without publishing either. The
	// persistent log is the commit point: only after it acknowledges the event
	// do we mirror into the in-memory journal and commit the run, so a durable
	// write failure leaves no orphan run to list or operate on.
	created := run.New(station)
	event := model.Event{ID: created.ID + "/loaded", RunID: created.ID, Kind: "state", Epoch: created.Epoch, Payload: string(created.State), At: created.CreatedAt}
	if err := r.commitEvent(event); err != nil {
		return model.Run{}, fmt.Errorf("journal: %w", err)
	}
	if err := r.Runs.Commit(created); err != nil {
		return model.Run{}, fmt.Errorf("commit run: %w", err)
	}
	return created, nil
}
func (r *Runtime) Operations() []model.Run { return run.Active(r.Runs) }
func (r *Runtime) JournalHealthy() bool    { return journal.Healthy(r.Journal) }

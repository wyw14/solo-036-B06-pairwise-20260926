package service

import (
	"fmt"
	"github.com/wyw14/solo-036/internal/journal"
	"github.com/wyw14/solo-036/internal/model"
	"github.com/wyw14/solo-036/internal/run"
	"time"
)

func (r *Runtime) CreateOperation(station string) (model.Run, error) {
	created := run.Prepare(station)
	event := model.Event{ID: created.ID + "/loaded", RunID: created.ID, Kind: "state", Epoch: created.Epoch, Payload: string(created.State), At: time.Now().UTC()}
	if err := r.commitCreation(created, event); err != nil {
		return model.Run{}, fmt.Errorf("journal: %w", err)
	}
	return created, nil
}
func (r *Runtime) Operations() []model.Run { return run.Active(r.Runs) }
func (r *Runtime) JournalHealthy() bool    { return journal.Healthy(r.Journal) }

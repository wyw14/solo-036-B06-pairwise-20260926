package run

import (
	"fmt"
	"github.com/wyw14/solo-036/internal/model"
	"time"
)

// New prepares a run without publishing it to the store. The caller owns the
// returned run and must only publish it through Store.Commit once the run is
// durably recorded, so a failed persistence attempt leaves no observable run.
func New(station string) model.Run {
	now := time.Now().UTC()
	r := model.NewRun(model.NewRevision(), station, 1)
	r.CreatedAt = now
	r.UpdatedAt = now
	return r
}

// Commit publishes a prepared run. It is the in-memory commit point and must
// only be called after the run's durable record has been confirmed.
func (s *Store) Commit(r model.Run) error {
	if !model.ValidateRun(r) {
		return fmt.Errorf("run %s is not valid for commit", r.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[r.ID]; exists {
		return fmt.Errorf("run %s already exists", r.ID)
	}
	s.runs[r.ID] = r
	return nil
}
func (s *Store) Get(id string) (model.Run, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	return r, ok
}
func (s *Store) Transition(id string, next model.RunState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return fmt.Errorf("run %s not found", id)
	}
	if !model.ValidTransition(r.State, next) {
		return fmt.Errorf("invalid transition %s to %s", r.State, next)
	}
	r.State = next
	r.Epoch++
	r.UpdatedAt = time.Now().UTC()
	s.runs[id] = r
	return nil
}
func (s *Store) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return fmt.Errorf("run %s not found", id)
	}
	if r.State.Terminal() {
		return nil
	}
	r.State = model.StateCancelled
	r.Epoch++
	r.UpdatedAt = time.Now().UTC()
	s.runs[id] = r
	return nil
}
func (s *Store) Snapshot() []model.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Run, 0, len(s.runs))
	for _, r := range s.runs {
		out = append(out, r)
	}
	return out
}

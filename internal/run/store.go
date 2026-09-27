package run

import (
	"fmt"
	"github.com/wyw14/solo-036/internal/model"
	"time"
)

// Prepare builds a run value without publishing it. The caller must durably
// commit the creation event before handing the value to Commit.
func Prepare(station string) model.Run {
	now := time.Now().UTC()
	r := model.NewRun(model.NewRevision(), station, 1)
	r.CreatedAt = now
	r.UpdatedAt = now
	return r
}

// Commit publishes a run after its creation event has been durably committed.
func (s *Store) Commit(r model.Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.ID] = r
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

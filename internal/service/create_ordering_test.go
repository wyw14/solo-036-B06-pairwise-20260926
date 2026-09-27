package service

import (
	"os"
	"path/filepath"
	"testing"
)

func newPersistedRuntime(t *testing.T, dir string) *Runtime {
	t.Helper()
	r := NewRuntime()
	if err := r.ConfigurePersistence(dir); err != nil {
		t.Fatalf("configure persistence: %v", err)
	}
	return r
}

func TestCreate_DurableFailureLeavesNoOrphan(t *testing.T) {
	dir := t.TempDir()
	r := newPersistedRuntime(t, dir)

	// Make the durable event log unopenable for append: events.jsonl as a dir.
	if err := os.Mkdir(filepath.Join(dir, "events.jsonl"), 0o750); err != nil {
		t.Fatalf("sabotage events path: %v", err)
	}

	if _, err := r.CreateOperation("station-a"); err == nil {
		t.Fatal("expected CreateOperation to fail when durable append fails")
	}
	if ops := r.Operations(); len(ops) != 0 {
		t.Fatalf("expected no live runs after durable failure, got %d: %+v", len(ops), ops)
	}
	if events := r.Journal.Events(); len(events) != 0 {
		t.Fatalf("expected no in-memory journal events after durable failure, got %d", len(events))
	}
	if n := len(r.Runs.Snapshot()); n != 0 {
		t.Fatalf("expected empty run store after durable failure, got %d", n)
	}
}

func TestCreate_SuccessRecoverableFromLog(t *testing.T) {
	dir := t.TempDir()
	r := newPersistedRuntime(t, dir)

	created, err := r.CreateOperation("station-a")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ops := r.Operations(); len(ops) != 1 || ops[0].ID != created.ID {
		t.Fatalf("expected the created run to be live, got %+v", ops)
	}

	// The creation must exist durably in the event log.
	persistent := r.control.Persistent
	events, err := persistent.Events()
	if err != nil {
		t.Fatalf("read durable events: %v", err)
	}
	if len(events) != 1 || events[0].RunID != created.ID {
		t.Fatalf("expected one durable creation event for %s, got %+v", created.ID, events)
	}
}

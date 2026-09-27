package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wyw14/solo-036/internal/model"
)

// failPersistentLog turns events.jsonl into a directory so the durable append
// fails deterministically with EISDIR, including when tests run as root (root
// bypasses permission bits but cannot open a directory as a writable file).
func failPersistentLog(t *testing.T, directory string) string {
	t.Helper()
	eventsPath := filepath.Join(directory, "events.jsonl")
	if err := os.MkdirAll(eventsPath, 0o750); err != nil {
		t.Fatalf("break event log: %v", err)
	}
	return eventsPath
}

func TestCreateOperationPersistentFailureLeavesNoOrphan(t *testing.T) {
	runtime := NewRuntime()
	directory := t.TempDir()
	if err := runtime.ConfigurePersistence(directory); err != nil {
		t.Fatalf("configure persistence: %v", err)
	}
	eventsPath := failPersistentLog(t, directory)

	if _, err := runtime.CreateOperation("station-a"); err == nil {
		t.Fatal("expected persistence failure, got nil error")
	}

	// No orphan run may be visible in the run list or operable by id.
	if operations := runtime.Operations(); len(operations) != 0 {
		t.Fatalf("expected no active runs after failed create, got %d: %+v", len(operations), operations)
	}
	if runs := runtime.Runs.Snapshot(); len(runs) != 0 {
		t.Fatalf("expected empty run store after failed create, got %d: %+v", len(runs), runs)
	}
	// The in-memory journal must not carry an event the durable log rejected.
	if events := runtime.Journal.Events(); len(events) != 0 {
		t.Fatalf("expected no in-memory journal events after failed create, got %d: %+v", len(events), events)
	}
	// The durable log tail must stay unchanged, leaving a valid empty chain.
	if sequence, checksum := runtime.control.Persistent.Tail(); sequence != 0 || checksum != "" {
		t.Fatalf("expected untouched durable tail, got sequence=%d checksum=%q", sequence, checksum)
	}

	// Once the fault clears, creation must still succeed and be recoverable.
	if err := os.RemoveAll(eventsPath); err != nil {
		t.Fatalf("repair event log: %v", err)
	}
	created, err := runtime.CreateOperation("station-a")
	if err != nil {
		t.Fatalf("retry create after fault cleared: %v", err)
	}
	if _, ok := runtime.Runs.Get(created.ID); !ok {
		t.Fatalf("created run %s is not committed to the store", created.ID)
	}
	if events := runtime.Journal.Events(); len(events) != 1 || events[0].RunID != created.ID {
		t.Fatalf("expected one in-memory event for %s, got %+v", created.ID, events)
	}
	persistentEvents, err := runtime.control.Persistent.Events()
	if err != nil {
		t.Fatalf("read persistent events: %v", err)
	}
	if len(persistentEvents) != 1 {
		t.Fatalf("expected one durable event, got %d", len(persistentEvents))
	}
	if !persistentEvents[0].Durable || persistentEvents[0].RunID != created.ID {
		t.Fatalf("durable event mismatch: %+v", persistentEvents[0])
	}
	if sequence, _ := runtime.control.Persistent.Tail(); sequence != 1 {
		t.Fatalf("expected durable sequence 1, got %d", sequence)
	}

	if err := runtime.PersistSnapshot(); err != nil {
		t.Fatalf("persist snapshot: %v", err)
	}
	recovered := NewRuntime()
	if err := recovered.ConfigurePersistence(directory); err != nil {
		t.Fatalf("configure recovered runtime: %v", err)
	}
	report, err := recovered.RecoverPersistent()
	if err != nil {
		t.Fatalf("recover persistent state: %v", err)
	}
	if !report.Ready {
		t.Fatal("expected recovered runtime to be ready")
	}
	restored, ok := recovered.Runs.Get(created.ID)
	if !ok {
		t.Fatalf("run %s was not recovered from the persisted log", created.ID)
	}
	if restored.State != model.StateLoaded || restored.Epoch != created.Epoch {
		t.Fatalf("recovered run mismatch: %+v", restored)
	}
	if recoveredEvents, err := recovered.control.Persistent.Events(); err != nil || len(recoveredEvents) != 1 {
		t.Fatalf("expected recovered log to retain one event, got %d (err=%v)", len(recoveredEvents), err)
	}
}

func TestCreateOperationWithoutPersistenceCommitsToMemoryJournal(t *testing.T) {
	runtime := NewRuntime()
	created, err := runtime.CreateOperation("station-a")
	if err != nil {
		t.Fatalf("create without persistence: %v", err)
	}
	if _, ok := runtime.Runs.Get(created.ID); !ok {
		t.Fatalf("created run %s is not committed", created.ID)
	}
	if events := runtime.Journal.Events(); len(events) != 1 || events[0].RunID != created.ID {
		t.Fatalf("expected one in-memory event for %s, got %+v", created.ID, events)
	}
}

package logger

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func TestMain(m *testing.M) {
	// Silence console output during the tests.
	log.SetOutput(io.Discard)
	code := m.Run()
	log.SetOutput(os.Stderr)
	os.Exit(code)
}

// recordingStore captures system log entries so we can assert they are persisted.
type recordingStore struct {
	entries []entry
}

type entry struct {
	guildID  string
	logType  model.SystemLogType
	function string
	content  string
}

func (r *recordingStore) AddSystemLogEntry(guildID string, logType model.SystemLogType, function, content string) error {
	r.entries = append(r.entries, entry{guildID, logType, function, content})
	return nil
}

func TestLogPersistsWhenDatabaseSet(t *testing.T) {
	l := NewLogger()
	store := &recordingStore{}

	l.LogInfo(LogModel{Database: store, GuildID: "g1", Function: "fn", Message: "hi"})
	l.LogError(LogModel{Database: store, GuildID: "g1", Function: "fn", Message: "boom"})
	l.LogWarning(LogModel{Database: store, GuildID: "g1", Function: "fn", Message: "warn"})
	l.LogSuccess(LogModel{Database: store, GuildID: "g1", Function: "fn", Message: "ok"})

	if len(store.entries) != 4 {
		t.Fatalf("expected 4 persisted entries, got %d", len(store.entries))
	}

	wantTypes := []model.SystemLogType{model.TypeInfo, model.TypeError, model.TypeWarning, model.TypeSuccess}
	for i, want := range wantTypes {
		if store.entries[i].logType != want {
			t.Errorf("entry %d type = %q, want %q", i, store.entries[i].logType, want)
		}
	}
}

func TestLogSkipsPersistenceWhenDatabaseNil(t *testing.T) {
	l := NewLogger()
	// Must not panic when no store is provided.
	l.LogInfo(LogModel{Message: "console only"})
	l.LogError(LogModel{Message: "console only"})
	l.LogWarning(LogModel{Message: "console only"})
	l.LogSuccess(LogModel{Message: "console only"})
}

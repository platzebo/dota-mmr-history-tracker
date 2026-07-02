package store

import (
	"path/filepath"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

func TestKnownMatchIDsReturnsStoredIDs(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.UpsertMatches([]ledger.Record{{MatchID: 11, StartTime: 1, MMRBefore: 100, RankChange: 25, MMRAfter: 125}, {MatchID: 22, StartTime: 2, MMRBefore: 125, RankChange: -25, MMRAfter: 100}}); err != nil {
		t.Fatal(err)
	}
	known, err := s.KnownMatchIDs()
	if err != nil {
		t.Fatal(err)
	}
	if !known[11] || !known[22] || known[33] || len(known) != 2 {
		t.Fatalf("unexpected known ids: %+v", known)
	}
}

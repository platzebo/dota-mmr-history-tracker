package store

import (
	"path/filepath"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

func TestSQLiteStoreUpsertsAndOrdersMatches(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	input := []ledger.Record{
		{MatchID: 2, StartTime: 200, HeroID: 2, MMRBefore: 3025, RankChange: -25, MMRAfter: 3000},
		{MatchID: 1, StartTime: 100, HeroID: 1, MMRBefore: 3000, RankChange: 25, MMRAfter: 3025},
	}
	if err := s.UpsertMatches(input); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertMatches([]ledger.Record{{MatchID: 1, StartTime: 100, HeroID: 1, MMRBefore: 3000, RankChange: 30, MMRAfter: 3030}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListMatches()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].MatchID != 1 || got[0].RankChange != 30 || got[1].MatchID != 2 {
		t.Fatalf("bad rows: %+v", got)
	}
}

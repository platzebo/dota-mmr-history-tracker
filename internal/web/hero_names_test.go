package web

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
	"github.com/platzebo/dota-mmr-history-tracker/internal/store"
)

func TestMatchesAPIIncludesHeroName(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.UpsertMatches([]ledger.Record{{MatchID: 1, StartTime: 1, HeroID: 83, MMRBefore: 1000, RankChange: 25, MMRAfter: 1025}}); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	New(s).Handler().ServeHTTP(rr, httptest.NewRequest("GET", "/api/matches", nil))
	var rows []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["hero_name"] != "Treant Protector" {
		t.Fatalf("expected hero_name in API response, got %s", rr.Body.String())
	}
}

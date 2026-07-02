package syncer

import (
	"context"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

func TestFetchRankedHistoryWithStatsCanSkipInitialPages(t *testing.T) {
	c := &fakeHistoryClient{pages: [][]ledger.RawMatch{
		{{MatchID: 90, StartTime: 1700000900, PreviousRank: 3000, RankChange: 25}},
		{{MatchID: 80, StartTime: 1700000800, PreviousRank: 3025, RankChange: -25}},
		{{MatchID: 70, StartTime: 1700000700, PreviousRank: 3000, RankChange: 25}},
	}}

	got, stats, err := FetchRankedHistoryWithStatsAndSkip(context.Background(), c, 2, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MatchID != 70 {
		t.Fatalf("expected only records after skipped pages, got %+v", got)
	}
	if stats.Pages != 3 || stats.SkippedPages != 2 || stats.SkippedRawMatches != 2 || stats.RawMatches != 3 || stats.RankedRows != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

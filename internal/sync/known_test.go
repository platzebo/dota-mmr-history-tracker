package syncer

import (
	"context"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

func TestFetchRankedHistoryStopsAtKnownMatchID(t *testing.T) {
	c := &fakeHistoryClient{pages: [][]ledger.RawMatch{
		{{MatchID: 100, StartTime: 1700001000, PreviousRank: 3000, RankChange: 25}, {MatchID: 90, StartTime: 1700000900, PreviousRank: 3025, RankChange: -25}},
		{{MatchID: 80, StartTime: 1700000800, PreviousRank: 3000, RankChange: 25}, {MatchID: 70, StartTime: 1700000700, PreviousRank: 3025, RankChange: -25}},
		{{MatchID: 60, StartTime: 1700000600, PreviousRank: 3000, RankChange: 25}},
	}}

	got, stats, err := FetchRankedHistoryWithKnown(context.Background(), c, 100, 2, 0, map[uint64]bool{80: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].MatchID != 100 || got[1].MatchID != 90 {
		t.Fatalf("expected only records newer than known match, got %+v", got)
	}
	if !stats.HitKnownMatch || stats.KnownMatchID != 80 || stats.Pages != 2 || stats.RawMatches != 4 || stats.RankedRows != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestFetchRankedHistoryIncludesNewerRowsBeforeKnownInSamePage(t *testing.T) {
	c := &fakeHistoryClient{pages: [][]ledger.RawMatch{
		{{MatchID: 100, StartTime: 1700001000, PreviousRank: 3000, RankChange: 25}, {MatchID: 90, StartTime: 1700000900, PreviousRank: 3025, RankChange: -25}},
	}}

	got, stats, err := FetchRankedHistoryWithKnown(context.Background(), c, 100, 2, 0, map[uint64]bool{90: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MatchID != 100 {
		t.Fatalf("expected only match newer than known match, got %+v", got)
	}
	if !stats.HitKnownMatch || stats.KnownMatchID != 90 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

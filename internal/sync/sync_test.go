package syncer

import (
	"context"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

type fakeHistoryClient struct{ pages [][]ledger.RawMatch }

func (f *fakeHistoryClient) FetchPage(ctx context.Context, startAt uint64, limit uint32) ([]ledger.RawMatch, error) {
	if len(f.pages) == 0 {
		return nil, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func TestFetchRankedHistoryPaginatesUntilLimit(t *testing.T) {
	c := &fakeHistoryClient{pages: [][]ledger.RawMatch{
		{{MatchID: 30, StartTime: 1700000300, PreviousRank: 3000, RankChange: 25, SoloRank: true}, {MatchID: 20, StartTime: 1700000200, PreviousRank: 3025, RankChange: -25, SoloRank: true}},
		{{MatchID: 10, StartTime: 1700000100, PreviousRank: 3000, RankChange: 0, SoloRank: true}, {MatchID: 9, StartTime: 1700000090, PreviousRank: 3000, RankChange: 25, SoloRank: true}},
	}}
	got, stats, err := FetchRankedHistoryWithStats(context.Background(), c, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d got=%+v", len(got), got)
	}
	if got[0].MatchID != 30 || got[1].MatchID != 20 || got[2].MatchID != 9 {
		t.Fatalf("unexpected records: %+v", got)
	}
	if stats.Pages != 2 || stats.RawMatches != 4 || stats.RowsWithPreviousMMR != 4 || stats.RowsWithRankChange != 3 || stats.RankedRows != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

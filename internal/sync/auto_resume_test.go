package syncer

import (
	"context"
	"errors"
	"testing"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

type recordingHistoryClient struct {
	pages   [][]ledger.RawMatch
	errPage int
	starts  []uint64
}

func (f *recordingHistoryClient) FetchPage(ctx context.Context, startAt uint64, limit uint32) ([]ledger.RawMatch, error) {
	f.starts = append(f.starts, startAt)
	if f.errPage > 0 && len(f.starts) == f.errPage {
		return nil, errors.New("rate limited")
	}
	if len(f.pages) == 0 {
		return nil, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func TestFetchRankedHistoryStartsFromSavedCursor(t *testing.T) {
	c := &recordingHistoryClient{pages: [][]ledger.RawMatch{
		{{MatchID: 70, StartTime: 1700000070, PreviousRank: 3000, RankChange: 25}, {MatchID: 60, StartTime: 1700000060, PreviousRank: 3025, RankChange: -25}},
	}}

	got, stats, err := FetchRankedHistoryWithKnownFrom(context.Background(), c, 10, 2, 0, 80, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.starts) == 0 || c.starts[0] != 80 {
		t.Fatalf("expected first request to start at saved cursor 80, got starts=%v", c.starts)
	}
	if len(got) != 2 || got[0].MatchID != 70 || got[1].MatchID != 60 {
		t.Fatalf("unexpected rows: %+v", got)
	}
	if stats.NextStartAtMatchID != 60 {
		t.Fatalf("expected next cursor 60, got stats=%+v", stats)
	}
}

func TestFetchRankedHistoryReturnsPartialRecordsAndCursorOnPageError(t *testing.T) {
	c := &recordingHistoryClient{errPage: 2, pages: [][]ledger.RawMatch{
		{{MatchID: 100, StartTime: 1700000100, PreviousRank: 3000, RankChange: 25}, {MatchID: 90, StartTime: 1700000090, PreviousRank: 3025, RankChange: -25}},
	}}

	got, stats, err := FetchRankedHistoryWithKnownFrom(context.Background(), c, 10, 2, 0, 0, nil)
	if err == nil {
		t.Fatal("expected page error")
	}
	if len(got) != 2 || got[0].MatchID != 100 || got[1].MatchID != 90 {
		t.Fatalf("expected partial records from successful page, got %+v", got)
	}
	if stats.NextStartAtMatchID != 90 || stats.Pages != 1 {
		t.Fatalf("expected resumable cursor from last successful page, got stats=%+v", stats)
	}
}

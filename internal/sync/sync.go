package syncer

import (
	"context"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

type HistoryClient interface {
	FetchPage(ctx context.Context, startAt uint64, limit uint32) ([]ledger.RawMatch, error)
}

type Stats struct {
	Pages               int
	RawMatches          int
	RowsWithPreviousMMR int
	RowsWithRankChange  int
	RankedRows          int
	SkippedPages        int
	SkippedRawMatches   int
	HitKnownMatch       bool
	KnownMatchID        uint64
}

func FetchRankedHistory(ctx context.Context, c HistoryClient, limit, pageSize int) ([]ledger.Record, error) {
	records, _, err := FetchRankedHistoryWithStats(ctx, c, limit, pageSize)
	return records, err
}

func FetchRankedHistoryWithStats(ctx context.Context, c HistoryClient, limit, pageSize int) ([]ledger.Record, Stats, error) {
	return FetchRankedHistoryWithKnown(ctx, c, limit, pageSize, 0, nil)
}

func FetchRankedHistoryWithStatsAndSkip(ctx context.Context, c HistoryClient, limit, pageSize, skipPages int) ([]ledger.Record, Stats, error) {
	return FetchRankedHistoryWithKnown(ctx, c, limit, pageSize, skipPages, nil)
}

func FetchRankedHistoryWithKnown(ctx context.Context, c HistoryClient, limit, pageSize, skipPages int, known map[uint64]bool) ([]ledger.Record, Stats, error) {
	if pageSize <= 0 || pageSize > 20 {
		pageSize = 20
	}
	if limit <= 0 {
		limit = pageSize
	}
	if skipPages < 0 {
		skipPages = 0
	}
	out := make([]ledger.Record, 0, limit)
	var stats Stats
	var startAt uint64
	for len(out) < limit {
		raw, err := c.FetchPage(ctx, startAt, uint32(pageSize))
		if err != nil {
			return nil, stats, err
		}
		if len(raw) == 0 {
			break
		}
		stats.Pages++
		stats.RawMatches += len(raw)
		for _, m := range raw {
			if m.PreviousRank != 0 {
				stats.RowsWithPreviousMMR++
			}
			if m.RankChange != 0 {
				stats.RowsWithRankChange++
			}
		}
		startAt = raw[len(raw)-1].MatchID
		if stats.Pages <= skipPages {
			stats.SkippedPages++
			stats.SkippedRawMatches += len(raw)
			if len(raw) < pageSize {
				break
			}
			continue
		}

		candidate := raw
		if len(known) > 0 {
			for i, m := range raw {
				if known[m.MatchID] {
					stats.HitKnownMatch = true
					stats.KnownMatchID = m.MatchID
					candidate = raw[:i]
					break
				}
			}
		}

		records := ledger.NormalizeMatches(candidate)
		stats.RankedRows += len(records)
		for _, r := range records {
			if len(out) >= limit {
				break
			}
			out = append(out, r)
		}
		if stats.HitKnownMatch || len(raw) < pageSize {
			break
		}
	}
	return out, stats, nil
}

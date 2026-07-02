package ledger

import (
	"encoding/csv"
	"strings"
	"testing"
)

func TestNormalizeKeepsOnlyRankedMMRRows(t *testing.T) {
	records := NormalizeMatches([]RawMatch{
		{MatchID: 10, StartTime: 1700000000, HeroID: 1, PreviousRank: 3000, RankChange: 25, SoloRank: true, Winner: true},
		{MatchID: 11, StartTime: 1700000100, HeroID: 2, PreviousRank: 0, RankChange: 25},
		{MatchID: 12, StartTime: 1700000200, HeroID: 3, PreviousRank: 3000, RankChange: 0},
	})
	if len(records) != 1 {
		t.Fatalf("expected one ranked row, got %d", len(records))
	}
	got := records[0]
	if got.MatchID != 10 || got.MMRBefore != 3000 || got.RankChange != 25 || got.MMRAfter != 3025 || !got.SoloRank || !got.Winner {
		t.Fatalf("unexpected normalized record: %+v", got)
	}
}

func TestNormalizeDropsOldPartyMMRRows(t *testing.T) {
	records := NormalizeMatches([]RawMatch{
		{MatchID: 1, StartTime: 1565000000, PreviousRank: 3000, RankChange: 25, SoloRank: false},
		{MatchID: 2, StartTime: 1565107201, PreviousRank: 3025, RankChange: -25, SoloRank: false},
	})
	if len(records) != 1 || records[0].MatchID != 2 {
		t.Fatalf("expected only post party-MMR-removal record, got %+v", records)
	}
}

func TestSummaryComputesCurrentPeakAndTotals(t *testing.T) {
	s := Summarize([]Record{
		{MatchID: 1, StartTime: 100, MMRBefore: 3000, RankChange: 25, MMRAfter: 3025, SoloRank: true, HeroID: 1},
		{MatchID: 2, StartTime: 200, MMRBefore: 3025, RankChange: -50, MMRAfter: 2975, SoloRank: false, HeroID: 2},
		{MatchID: 3, StartTime: 300, MMRBefore: 2975, RankChange: 30, MMRAfter: 3005, SoloRank: true, HeroID: 1},
	})
	if s.CurrentMMR != 3005 || s.PeakMMR != 3025 || s.LowestMMR != 2975 || s.TotalChange != 5 || s.SoloChange != 55 || s.PartyChange != -50 {
		t.Fatalf("bad summary: %+v", s)
	}
	if s.HeroChange[1] != 55 || s.HeroChange[2] != -50 {
		t.Fatalf("bad hero totals: %+v", s.HeroChange)
	}
}

func TestCSVExportUsesExpectedColumns(t *testing.T) {
	out, err := ExportCSV([]Record{{MatchID: 99, StartTime: 1700000000, SoloRank: true, HeroID: 42, MMRBefore: 1234, RankChange: -25}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%v", rows)
	}
	wantHeader := []string{"Date", "Unix time", "MatchID", "Solo Queue", "HeroID", "Start MMR", "Rank Change"}
	for i := range wantHeader {
		if rows[0][i] != wantHeader[i] {
			t.Fatalf("header[%d]=%q", i, rows[0][i])
		}
	}
	if rows[1][2] != "99" || rows[1][3] != "true" || rows[1][4] != "42" || rows[1][5] != "1234" || rows[1][6] != "-25" {
		t.Fatalf("bad row: %v", rows[1])
	}
}

package ledger

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"time"
)

const partyMMRRemovalUnix uint32 = 1565049600 // 2019-08-06 00:00:00 UTC

type RawMatch struct {
	MatchID                uint64 `json:"match_id"`
	StartTime              uint32 `json:"start_time"`
	HeroID                 int32  `json:"hero_id"`
	Winner                 bool   `json:"winner"`
	GameMode               uint32 `json:"game_mode"`
	RankChange             int32  `json:"rank_change"`
	PreviousRank           uint32 `json:"previous_rank"`
	LobbyType              uint32 `json:"lobby_type"`
	SoloRank               bool   `json:"solo_rank"`
	Abandon                bool   `json:"abandon"`
	Duration               uint32 `json:"duration"`
	Engine                 uint32 `json:"engine"`
	ActivePlusSubscription bool   `json:"active_plus_subscription"`
	SeasonalRank           bool   `json:"seasonal_rank"`
	SelectedFacet          uint32 `json:"selected_facet"`
}

type Record struct {
	MatchID    uint64 `json:"match_id"`
	StartTime  uint32 `json:"start_time"`
	HeroID     int32  `json:"hero_id"`
	Winner     bool   `json:"winner"`
	GameMode   uint32 `json:"game_mode"`
	LobbyType  uint32 `json:"lobby_type"`
	SoloRank   bool   `json:"solo_rank"`
	Abandon    bool   `json:"abandon"`
	Duration   uint32 `json:"duration"`
	MMRBefore  int32  `json:"mmr_before"`
	RankChange int32  `json:"rank_change"`
	MMRAfter   int32  `json:"mmr_after"`
}

type Summary struct {
	MatchCount  int             `json:"match_count"`
	CurrentMMR  int32           `json:"current_mmr"`
	PeakMMR     int32           `json:"peak_mmr"`
	LowestMMR   int32           `json:"lowest_mmr"`
	TotalChange int32           `json:"total_change"`
	SoloChange  int32           `json:"solo_change"`
	PartyChange int32           `json:"party_change"`
	HeroChange  map[int32]int32 `json:"hero_change"`
}

func NormalizeMatches(raw []RawMatch) []Record {
	out := make([]Record, 0, len(raw))
	for _, m := range raw {
		if m.MatchID == 0 || m.StartTime == 0 || m.PreviousRank == 0 || m.RankChange == 0 {
			continue
		}
		if m.StartTime < partyMMRRemovalUnix && !m.SoloRank {
			continue
		}
		before := int32(m.PreviousRank)
		out = append(out, Record{
			MatchID: m.MatchID, StartTime: m.StartTime, HeroID: m.HeroID, Winner: m.Winner,
			GameMode: m.GameMode, LobbyType: m.LobbyType, SoloRank: m.SoloRank, Abandon: m.Abandon,
			Duration: m.Duration, MMRBefore: before, RankChange: m.RankChange, MMRAfter: before + m.RankChange,
		})
	}
	return out
}

func Summarize(records []Record) Summary {
	s := Summary{HeroChange: map[int32]int32{}}
	if len(records) == 0 {
		return s
	}
	sorted := append([]Record(nil), records...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartTime < sorted[j].StartTime })
	s.MatchCount = len(sorted)
	s.PeakMMR = sorted[0].MMRAfter
	s.LowestMMR = sorted[0].MMRAfter
	for _, r := range sorted {
		s.CurrentMMR = r.MMRAfter
		if r.MMRAfter > s.PeakMMR {
			s.PeakMMR = r.MMRAfter
		}
		if r.MMRAfter < s.LowestMMR {
			s.LowestMMR = r.MMRAfter
		}
		s.TotalChange += r.RankChange
		if r.SoloRank {
			s.SoloChange += r.RankChange
		} else {
			s.PartyChange += r.RankChange
		}
		s.HeroChange[r.HeroID] += r.RankChange
	}
	return s
}

func ExportCSV(records []Record) (string, error) {
	sorted := append([]Record(nil), records...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartTime < sorted[j].StartTime })
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"Date", "Unix time", "MatchID", "Solo Queue", "HeroID", "Start MMR", "Rank Change"}); err != nil {
		return "", err
	}
	for _, r := range sorted {
		row := []string{
			time.Unix(int64(r.StartTime), 0).Local().Format("2006-01-02 15:04:05"),
			strconv.FormatUint(uint64(r.StartTime), 10),
			strconv.FormatUint(r.MatchID, 10),
			strconv.FormatBool(r.SoloRank),
			fmt.Sprintf("%d", r.HeroID),
			fmt.Sprintf("%d", r.MMRBefore),
			fmt.Sprintf("%d", r.RankChange),
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

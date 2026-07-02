package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS matches (
        match_id INTEGER PRIMARY KEY,
        start_time INTEGER NOT NULL,
        hero_id INTEGER NOT NULL,
        winner INTEGER NOT NULL,
        game_mode INTEGER NOT NULL,
        lobby_type INTEGER NOT NULL,
        solo_rank INTEGER NOT NULL,
        abandon INTEGER NOT NULL,
        duration INTEGER NOT NULL,
        mmr_before INTEGER NOT NULL,
        rank_change INTEGER NOT NULL,
        mmr_after INTEGER NOT NULL,
        imported_at INTEGER NOT NULL DEFAULT (unixepoch())
    );
    CREATE INDEX IF NOT EXISTS idx_matches_start_time ON matches(start_time);`)
	return err
}

func (s *Store) UpsertMatches(records []ledger.Record) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO matches (match_id,start_time,hero_id,winner,game_mode,lobby_type,solo_rank,abandon,duration,mmr_before,rank_change,mmr_after)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(match_id) DO UPDATE SET
          start_time=excluded.start_time, hero_id=excluded.hero_id, winner=excluded.winner, game_mode=excluded.game_mode,
          lobby_type=excluded.lobby_type, solo_rank=excluded.solo_rank, abandon=excluded.abandon, duration=excluded.duration,
          mmr_before=excluded.mmr_before, rank_change=excluded.rank_change, mmr_after=excluded.mmr_after`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range records {
		if _, err := stmt.Exec(r.MatchID, r.StartTime, r.HeroID, boolInt(r.Winner), r.GameMode, r.LobbyType, boolInt(r.SoloRank), boolInt(r.Abandon), r.Duration, r.MMRBefore, r.RankChange, r.MMRAfter); err != nil {
			return fmt.Errorf("upsert match %d: %w", r.MatchID, err)
		}
	}
	return tx.Commit()
}

func (s *Store) ListMatches() ([]ledger.Record, error) {
	rows, err := s.db.Query(`SELECT match_id,start_time,hero_id,winner,game_mode,lobby_type,solo_rank,abandon,duration,mmr_before,rank_change,mmr_after FROM matches ORDER BY start_time ASC, match_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ledger.Record, 0)
	for rows.Next() {
		var r ledger.Record
		var winner, solo, abandon int
		if err := rows.Scan(&r.MatchID, &r.StartTime, &r.HeroID, &winner, &r.GameMode, &r.LobbyType, &solo, &abandon, &r.Duration, &r.MMRBefore, &r.RankChange, &r.MMRAfter); err != nil {
			return nil, err
		}
		r.Winner = winner != 0
		r.SoloRank = solo != 0
		r.Abandon = abandon != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) KnownMatchIDs() (map[uint64]bool, error) {
	rows, err := s.db.Query(`SELECT match_id FROM matches`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uint64]bool)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
	"github.com/platzebo/dota-mmr-history-tracker/internal/steamgc"
	"github.com/platzebo/dota-mmr-history-tracker/internal/store"
	webui "github.com/platzebo/dota-mmr-history-tracker/internal/web"
)

type stdoutPrompter struct{}

func (stdoutPrompter) Printf(format string, args ...any) { fmt.Printf(format, args...) }

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "sync":
		runSync(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usageText() string {
	return `dota-mmr-history-tracker

Automatic Dota 2 MMR history tracker using Steam/Dota GameCoordinator.
No manual MMR entry path is implemented.

Commands:
  sync   --username USER [--auto] [--qr | --access-token TOKEN] [--matches 500] [--skip-pages N] [--dump-raw raw.jsonl]
  serve  [--db PATH] [--addr 127.0.0.1:8789]
  export [--db PATH] [--out FILE]

Environment fallbacks:
  STEAM_USERNAME, STEAM_ACCESS_TOKEN

`
}

func usage() {
	fmt.Fprint(os.Stderr, usageText())
}

func defaultDB() string {
	if v := os.Getenv("DOTA_MMR_HISTORY_TRACKER_DB"); v != "" {
		return v
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "dota-mmr-history-tracker", "ledger.sqlite")
}

func openStore(path string) *store.Store {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	s, err := store.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func runSync(args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB(), "SQLite database path")
	username := fs.String("username", getenv("STEAM_USERNAME"), "Steam username")
	accessToken := fs.String("access-token", getenv("STEAM_ACCESS_TOKEN"), "Steam access token")
	useQR := fs.Bool("qr", false, "authenticate by scanning a Steam mobile QR code")
	matches := fs.Int("matches", 500, "ranked MMR rows to fetch")
	rawDump := fs.String("dump-raw", "", "write raw Dota GC match-history rows as JSONL for debugging")
	autoSync := fs.Bool("auto", false, "automatic sync: import newest rows or resume older history from the saved cursor")
	skipPages := fs.Int("skip-pages", 0, "skip this many initial 20-match GC pages before importing; useful for backfilling older ranges")
	pageDelay := fs.Duration("page-delay", time.Second, "delay between 20-match GC pages to avoid rate limits")
	timeout := fs.Duration("timeout", 3*time.Minute, "sync timeout")
	fs.Parse(args)

	var known map[uint64]bool
	var startAt uint64
	s := openStore(*dbPath)
	defer s.Close()
	if *autoSync {
		var err error
		known, err = s.KnownMatchIDs()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("auto sync: loaded %d known match IDs from %s\n", len(known), *dbPath)
		if cursor, ok, err := s.AutoBackfillCursor(); err != nil {
			log.Fatal(err)
		} else if ok && *skipPages == 0 {
			startAt = cursor
			known = nil
			fmt.Printf("auto sync: resuming older history from match_id=%d\n", cursor)
		}
	}

	report, err := steamgc.FetchReport(context.Background(), steamgc.Options{Username: *username, AccessToken: *accessToken, UseQR: *useQR, Limit: *matches, Timeout: *timeout, Prompter: stdoutPrompter{}, RawDumpPath: *rawDump, PageDelay: *pageDelay, SkipPages: *skipPages, StartAtMatchID: startAt, KnownMatchIDs: known})
	records := report.Records
	if len(records) > 0 {
		if upsertErr := s.UpsertMatches(records); upsertErr != nil {
			log.Fatal(upsertErr)
		}
	}
	if *autoSync {
		if report.Stats.Exhausted {
			if cursorErr := s.ClearAutoBackfillCursor(); cursorErr != nil {
				log.Fatal(cursorErr)
			}
			fmt.Println("auto sync: reached end of available history; resume cursor cleared")
		} else if !report.Stats.HitKnownMatch && report.Stats.NextStartAtMatchID != 0 {
			if cursorErr := s.SetAutoBackfillCursor(report.Stats.NextStartAtMatchID); cursorErr != nil {
				log.Fatal(cursorErr)
			}
			fmt.Printf("auto sync: saved resume cursor match_id=%d\n", report.Stats.NextStartAtMatchID)
		}
	}
	if err != nil {
		fmt.Printf("synced %d ranked MMR rows into %s before error\n", len(records), *dbPath)
		fmt.Printf("gc_scan pages=%d raw_matches=%d skipped_pages=%d skipped_raw_matches=%d with_previous_mmr=%d with_rank_change=%d ranked_rows=%d hit_known=%t known_match_id=%d next_start_at_match_id=%d exhausted=%t\n", report.Stats.Pages, report.Stats.RawMatches, report.Stats.SkippedPages, report.Stats.SkippedRawMatches, report.Stats.RowsWithPreviousMMR, report.Stats.RowsWithRankChange, report.Stats.RankedRows, report.Stats.HitKnownMatch, report.Stats.KnownMatchID, report.Stats.NextStartAtMatchID, report.Stats.Exhausted)
		log.Fatal(err)
	}
	summary := ledger.Summarize(records)
	fmt.Printf("synced %d ranked MMR rows into %s\n", len(records), *dbPath)
	fmt.Printf("gc_scan pages=%d raw_matches=%d skipped_pages=%d skipped_raw_matches=%d with_previous_mmr=%d with_rank_change=%d ranked_rows=%d hit_known=%t known_match_id=%d next_start_at_match_id=%d exhausted=%t\n", report.Stats.Pages, report.Stats.RawMatches, report.Stats.SkippedPages, report.Stats.SkippedRawMatches, report.Stats.RowsWithPreviousMMR, report.Stats.RowsWithRankChange, report.Stats.RankedRows, report.Stats.HitKnownMatch, report.Stats.KnownMatchID, report.Stats.NextStartAtMatchID, report.Stats.Exhausted)
	fmt.Printf("current=%d peak=%d lowest=%d total_delta=%+d\n", summary.CurrentMMR, summary.PeakMMR, summary.LowestMMR, summary.TotalChange)
	if len(records) == 0 {
		if report.Stats.RawMatches == 0 {
			fmt.Println("diagnostic: Dota GC returned no match-history rows for this account/request.")
		} else {
			fmt.Println("diagnostic: Dota GC returned matches, but none contained both previous_rank and rank_change. This usually means no calibrated/ranked MMR history is available for the account yet, or the recent history has no MMR-changing matches.")
		}
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB(), "SQLite database path")
	addr := fs.String("addr", "127.0.0.1:8789", "HTTP listen address")
	fs.Parse(args)
	s := openStore(*dbPath)
	defer s.Close()
	srv := &http.Server{Addr: *addr, Handler: webui.New(s).Handler(), ReadHeaderTimeout: 5 * time.Second}
	fmt.Printf("serving http://%s from %s\n", *addr, *dbPath)
	log.Fatal(srv.ListenAndServe())
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB(), "SQLite database path")
	outPath := fs.String("out", "", "output CSV path, stdout if empty")
	fs.Parse(args)
	s := openStore(*dbPath)
	defer s.Close()
	rows, err := s.ListMatches()
	if err != nil {
		log.Fatal(err)
	}
	out, err := ledger.ExportCSV(rows)
	if err != nil {
		log.Fatal(err)
	}
	if *outPath == "" {
		fmt.Print(out)
		return
	}
	if err := os.WriteFile(*outPath, []byte(out), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s\n", *outPath)
}

func getenv(k string) string { return os.Getenv(k) }

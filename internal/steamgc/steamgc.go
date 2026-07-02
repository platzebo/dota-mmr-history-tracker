package steamgc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dota2 "github.com/paralin/go-dota2"
	dotaevents "github.com/paralin/go-dota2/events"
	"github.com/paralin/go-dota2/protocol"
	steam "github.com/paralin/go-steam"
	"github.com/paralin/go-steam/protocol/steamlang"
	"github.com/sirupsen/logrus"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
	"github.com/platzebo/dota-mmr-history-tracker/internal/steamauth"
	syncer "github.com/platzebo/dota-mmr-history-tracker/internal/sync"
)

type Options struct {
	Username      string
	AccessToken   string
	UseQR         bool
	Limit         int
	Timeout       time.Duration
	Prompter      steamauth.QRPrompter
	RawDumpPath   string
	PageDelay     time.Duration
	SkipPages     int
	KnownMatchIDs map[uint64]bool
	QRTokenFunc   func(context.Context, steamauth.QRAuthOptions) (string, error)
}

type Report struct {
	Records []ledger.Record
	Stats   syncer.Stats
}

func logf(prompter steamauth.QRPrompter, format string, args ...any) {
	if prompter != nil {
		prompter.Printf(format, args...)
	}
}

type pageClient struct {
	dota      *dota2.Dota2
	accountID uint32
	page      int
	dump      *json.Encoder
	delay     time.Duration
	logf      func(string, ...any)
}

func (p *pageClient) FetchPage(ctx context.Context, startAt uint64, limit uint32) ([]ledger.RawMatch, error) {
	p.page++
	if p.logf != nil {
		p.logf("[sync] request page=%d start_at_match_id=%d limit=%d\n", p.page, startAt, limit)
	}
	req := &protocol.CMsgDOTAGetPlayerMatchHistory{
		AccountId:        &p.accountID,
		MatchesRequested: &limit,
	}
	if startAt != 0 {
		req.StartAtMatchId = &startAt
	}
	resp, err := p.dota.GetPlayerMatchHistory(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]ledger.RawMatch, 0, len(resp.GetMatches()))
	withPrevious := 0
	withDelta := 0
	var newest, oldest uint64
	for i, m := range resp.GetMatches() {
		raw := ledger.RawMatch{
			MatchID: m.GetMatchId(), StartTime: m.GetStartTime(), HeroID: m.GetHeroId(), Winner: m.GetWinner(),
			GameMode: m.GetGameMode(), RankChange: m.GetRankChange(), PreviousRank: m.GetPreviousRank(),
			LobbyType: m.GetLobbyType(), SoloRank: m.GetSoloRank(), Abandon: m.GetAbandon(), Duration: m.GetDuration(),
			Engine: m.GetEngine(), ActivePlusSubscription: m.GetActivePlusSubscription(), SeasonalRank: m.GetSeasonalRank(), SelectedFacet: m.GetSelectedFacet(),
		}
		if i == 0 {
			newest = raw.MatchID
		}
		oldest = raw.MatchID
		if raw.PreviousRank != 0 {
			withPrevious++
		}
		if raw.RankChange != 0 {
			withDelta++
		}
		if p.dump != nil {
			_ = p.dump.Encode(raw)
		}
		out = append(out, raw)
	}
	if p.logf != nil {
		p.logf("[sync] page=%d received=%d newest=%d oldest=%d with_previous_mmr=%d with_rank_change=%d next_start=%d\n", p.page, len(out), newest, oldest, withPrevious, withDelta, oldest)
	}
	if len(out) > 0 && p.delay > 0 {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(p.delay):
		}
	}
	return out, nil
}

func Fetch(ctx context.Context, opts Options) ([]ledger.Record, error) {
	report, err := FetchReport(ctx, opts)
	if err != nil {
		return nil, err
	}
	return report.Records, nil
}

func FetchReport(ctx context.Context, opts Options) (Report, error) {
	if opts.Username == "" {
		return Report{}, errors.New("steam username is required")
	}
	if opts.AccessToken == "" && !opts.UseQR {
		return Report{}, errors.New("Steam access token or QR auth is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 500
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 3 * time.Minute
	}
	if opts.PageDelay == 0 {
		opts.PageDelay = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var dumpFile *os.File
	var dumpEncoder *json.Encoder
	if opts.RawDumpPath != "" {
		if err := os.MkdirAll(filepath.Dir(opts.RawDumpPath), 0o755); err != nil {
			return Report{}, err
		}
		var err error
		dumpFile, err = os.Create(opts.RawDumpPath)
		if err != nil {
			return Report{}, err
		}
		defer dumpFile.Close()
		dumpEncoder = json.NewEncoder(dumpFile)
		logf(opts.Prompter, "[sync] raw dump: %s\n", opts.RawDumpPath)
	}

	accessToken := opts.AccessToken
	if accessToken == "" && opts.UseQR {
		logf(opts.Prompter, "[auth] starting Steam QR authentication\n")
		qrTokenFunc := opts.QRTokenFunc
		if qrTokenFunc == nil {
			qrTokenFunc = steamauth.GetTokenViaQR
		}
		var err error
		accessToken, err = qrTokenFunc(ctx, steamauth.QRAuthOptions{DeviceName: "dota-mmr-history-tracker", Timeout: opts.Timeout, Prompter: opts.Prompter})
		if err != nil {
			return Report{}, fmt.Errorf("Steam QR auth: %w", err)
		}
		logf(opts.Prompter, "[auth] QR confirmed; received Steam token\n")
	}

	client := steam.NewClient()
	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
	d2 := dota2.New(client, log)
	resultCh := make(chan struct {
		report Report
		err    error
	}, 1)
	var accountID uint32

	logf(opts.Prompter, "[steam] connecting to Steam CM...\n")
	client.Connect()
	for {
		select {
		case <-ctx.Done():
			client.Disconnect()
			return Report{}, ctx.Err()
		case result := <-resultCh:
			client.Disconnect()
			return result.report, result.err
		case event := <-client.Events():
			switch e := event.(type) {
			case *steam.ConnectedEvent:
				logf(opts.Prompter, "[steam] connected; logging in as %s...\n", opts.Username)
				err := client.Auth.LogOn(ctx, &steam.LogOnDetails{
					Username: opts.Username, AccessToken: accessToken,
					DeviceFriendlyName: "dota-mmr-history-tracker", ShouldRememberPassword: true,
				})
				if err != nil {
					return Report{}, fmt.Errorf("steam auth: %w", err)
				}
			case *steam.LoggedOnEvent:
				if e.Result != steamlang.EResult_OK {
					return Report{}, fmt.Errorf("steam logon failed: %s", e.Result.String())
				}
				accountID = e.ClientSteamId.GetAccountId()
				logf(opts.Prompter, "[steam] logged in; account_id=%d\n", accountID)
				logf(opts.Prompter, "[dota] setting AppID 570 as playing and waiting for GameCoordinator...\n")
				d2.SetPlaying(true)
				time.AfterFunc(3*time.Second, func() { d2.SayHello() })
			case *steam.LogOnFailedEvent:
				if e.Err != nil {
					return Report{}, fmt.Errorf("steam logon failed state=%s: %w", e.AuthSessionState, e.Err)
				}
				return Report{}, fmt.Errorf("steam logon failed: %s", e.Result.String())
			case *steam.SteamFailureEvent:
				return Report{}, fmt.Errorf("steam failure: %s", e.Result.String())
			case *dotaevents.ClientWelcomed:
				logf(opts.Prompter, "[dota] GameCoordinator ready; fetching up to %d history rows (20 per page, skip_pages=%d, known_ids=%d, delay=%s)\n", opts.Limit, opts.SkipPages, len(opts.KnownMatchIDs), opts.PageDelay)
				pc := &pageClient{dota: d2, accountID: accountID, dump: dumpEncoder, delay: opts.PageDelay, logf: func(format string, args ...any) { logf(opts.Prompter, format, args...) }}
				go func() {
					records, stats, err := syncer.FetchRankedHistoryWithKnown(ctx, pc, opts.Limit, 20, opts.SkipPages, opts.KnownMatchIDs)
					resultCh <- struct {
						report Report
						err    error
					}{Report{Records: records, Stats: stats}, err}
				}()
			case steam.FatalErrorEvent:
				return Report{}, fmt.Errorf("steam fatal error: %w", error(e))
			case error:
				return Report{}, e
			}
		}
	}
}

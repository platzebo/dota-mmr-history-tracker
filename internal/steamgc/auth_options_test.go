package steamgc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/platzebo/dota-mmr-history-tracker/internal/steamauth"
)

type noopPrompter struct{}

func (noopPrompter) Printf(string, ...any) {}

func TestFetchReportRequiresQRorAccessTokenOnly(t *testing.T) {
	_, err := FetchReport(context.Background(), Options{Username: "user", Timeout: time.Nanosecond, Prompter: noopPrompter{}})
	if err == nil || !strings.Contains(err.Error(), "Steam access token or QR auth is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchReportCanUseInjectedQRTokenProvider(t *testing.T) {
	_, err := FetchReport(context.Background(), Options{
		Username: "user",
		UseQR:    true,
		Timeout:  time.Nanosecond,
		Prompter: noopPrompter{},
		QRTokenFunc: func(context.Context, steamauth.QRAuthOptions) (string, error) {
			return "token", nil
		},
	})
	if err == nil || strings.Contains(err.Error(), "access token or QR") {
		t.Fatalf("expected to get past auth validation, got: %v", err)
	}
}

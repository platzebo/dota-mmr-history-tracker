package steamgc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/platzebo/dota-mmr-history-tracker/internal/steamauth"
)

type capturePrompter struct{}

func (capturePrompter) Printf(format string, args ...any) {}

func TestFetchRunsQRAuthBeforeSteamCMConnect(t *testing.T) {
	called := false
	_, err := Fetch(context.Background(), Options{
		Username: "alice",
		UseQR:    true,
		Limit:    1,
		Timeout:  10 * time.Millisecond,
		Prompter: capturePrompter{},
		QRTokenFunc: func(ctx context.Context, opts steamauth.QRAuthOptions) (string, error) {
			called = true
			if opts.DeviceName != "dota-mmr-history-tracker" {
				t.Fatalf("device name = %q", opts.DeviceName)
			}
			return "fake-token", nil
		},
	})
	if !called {
		t.Fatal("QR token function was not called")
	}
	if err == nil {
		t.Fatal("expected timeout/connection error after fake token")
	}
}

func TestFetchReturnsQRAuthErrorBeforeConnecting(t *testing.T) {
	want := errors.New("denied")
	_, err := Fetch(context.Background(), Options{
		Username: "alice",
		UseQR:    true,
		Timeout:  time.Second,
		QRTokenFunc: func(ctx context.Context, opts steamauth.QRAuthOptions) (string, error) {
			return "", want
		},
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected QR error %v, got %v", want, err)
	}
}

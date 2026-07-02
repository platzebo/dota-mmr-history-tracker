package steamauth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/paralin/go-steam/protocol/protobuf/unified"
	qrcode "github.com/skip2/go-qrcode"
)

const authServiceBaseURL = "https://api.steampowered.com/IAuthenticationService/"

type QRPrompter interface {
	Printf(format string, args ...any)
}

type QRAuthOptions struct {
	DeviceName string
	Timeout    time.Duration
	Prompter   QRPrompter
}

// GetTokenViaQR starts Steam's modern QR-code auth flow, prints a QR code / URL,
// waits for approval in the Steam mobile app, and returns the resulting token for CM logon.
func GetTokenViaQR(ctx context.Context, opts QRAuthOptions) (string, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	deviceName := opts.DeviceName
	if deviceName == "" {
		deviceName = "dota-mmr-history-tracker"
	}
	platform := unified.EAuthTokenPlatformType_k_EAuthTokenPlatformType_SteamClient
	beginResp := new(unified.CAuthentication_BeginAuthSessionViaQR_Response)
	if err := authServiceCall(ctx, http.MethodPost, "BeginAuthSessionViaQR", &unified.CAuthentication_BeginAuthSessionViaQR_Request{
		DeviceFriendlyName: stringPtr(deviceName),
		PlatformType:       &platform,
		DeviceDetails: &unified.CAuthentication_DeviceDetails{
			DeviceFriendlyName: stringPtr(deviceName),
			PlatformType:       &platform,
		},
	}, beginResp); err != nil {
		return "", err
	}

	challengeURL := beginResp.GetChallengeUrl()
	if challengeURL == "" {
		return "", errors.New("Steam QR auth did not return a challenge URL")
	}
	if opts.Prompter != nil {
		if qr, err := qrcode.New(challengeURL, qrcode.Medium); err == nil {
			opts.Prompter.Printf("%s\n", qr.ToSmallString(false))
		}
		opts.Prompter.Printf("Open/scan with Steam mobile app: %s\n", challengeURL)
		opts.Prompter.Printf("Waiting for Steam mobile confirmation...\n")
	}

	interval := time.Duration(beginResp.GetInterval() * float32(time.Second))
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		pollResp := new(unified.CAuthentication_PollAuthSessionStatus_Response)
		if err := authServiceCall(ctx, http.MethodPost, "PollAuthSessionStatus", &unified.CAuthentication_PollAuthSessionStatus_Request{
			ClientId:  uint64Ptr(beginResp.GetClientId()),
			RequestId: beginResp.GetRequestId(),
		}, pollResp); err != nil {
			return "", err
		}
		if token := pollResp.GetRefreshToken(); token != "" {
			return token, nil
		}
		if token := pollResp.GetAccessToken(); token != "" {
			return token, nil
		}
		if agreement := pollResp.GetAgreementSessionUrl(); agreement != "" {
			return "", fmt.Errorf("Steam QR auth requires agreement: %s", agreement)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}
}

type vtMessage interface {
	MarshalVT() ([]byte, error)
	UnmarshalVT([]byte) error
}

func authServiceCall(ctx context.Context, method, name string, req vtMessage, resp vtMessage) error {
	encoded, err := req.MarshalVT()
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("input_protobuf_encoded", base64.StdEncoding.EncodeToString(encoded))
	endpoint := authServiceBaseURL + name + "/v1/"
	var httpReq *http.Request
	if method == http.MethodGet {
		httpReq, err = http.NewRequestWithContext(ctx, method, endpoint+"?"+params.Encode(), nil)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(params.Encode()))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return err
	}
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	if eresult := httpResp.Header.Get("x-eresult"); eresult != "" && eresult != "1" {
		msg := httpResp.Header.Get("x-error_message")
		if msg == "" {
			msg = httpResp.Status
		}
		return fmt.Errorf("%s failed: eresult %s (%s)", name, eresult, msg)
	}
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s failed: %s: %s", name, httpResp.Status, string(body))
	}
	return resp.UnmarshalVT(body)
}

func stringPtr(v string) *string { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }

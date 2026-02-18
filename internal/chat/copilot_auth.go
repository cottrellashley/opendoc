package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	copilotClientID       = "Iv1.b507a08c87ecfe98"
	copilotDeviceCodeURL  = "https://github.com/login/device/code"
	copilotAccessTokenURL = "https://github.com/login/oauth/access_token"
	copilotScope          = "read:user"
)

// DeviceCodeResponse is returned when initiating the device flow.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// StartCopilotDeviceFlow initiates the GitHub OAuth device code flow.
func StartCopilotDeviceFlow() (*DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {copilotClientID},
		"scope":     {copilotScope},
	}

	req, err := http.NewRequest("POST", copilotDeviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("device code failed (%d): %s", resp.StatusCode, string(body))
	}

	var result DeviceCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Interval == 0 {
		result.Interval = 5
	}

	return &result, nil
}

// PollCopilotAccessToken polls GitHub for the OAuth access token after the user
// has approved the device flow. Returns the token or an error. This blocks for
// up to ~3 minutes with 5s intervals.
func PollCopilotAccessToken(deviceCode string, interval int) (string, error) {
	if interval < 5 {
		interval = 5
	}

	maxAttempts := 36 // ~3 minutes at 5s intervals
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			time.Sleep(time.Duration(interval) * time.Second)
		}

		data := url.Values{
			"client_id":   {copilotClientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}

		req, err := http.NewRequest("POST", copilotAccessTokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.AccessToken != "" {
			return result.AccessToken, nil
		}

		switch result.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5
			continue
		case "expired_token":
			return "", fmt.Errorf("device code expired — please try again")
		case "access_denied":
			return "", fmt.Errorf("authorization denied by user")
		default:
			if result.Error != "" {
				return "", fmt.Errorf("oauth error: %s", result.Error)
			}
		}
	}

	return "", fmt.Errorf("timed out waiting for authorization")
}

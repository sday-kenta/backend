package pushclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var ErrUnregisteredToken = errors.New("push token is no longer registered")

type Message struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

type Sender interface {
	Send(context.Context, Message) error
}

type noopSender struct{}

func NewNoopSender() Sender {
	return noopSender{}
}

func (noopSender) Send(context.Context, Message) error {
	return nil
}

type Config struct {
	CredentialsFile string
	Timeout         time.Duration
}

type FCMSender struct {
	endpoint   string
	httpClient *http.Client
}

func NewFCMSender(ctx context.Context, cfg Config) (Sender, error) {
	path := strings.TrimSpace(cfg.CredentialsFile)
	if path == "" {
		return nil, errors.New("fcm credentials file is required")
	}

	rawCredentials, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fcm credentials file: %w", err)
	}

	var credentials serviceAccountCredentials
	if err = json.Unmarshal(rawCredentials, &credentials); err != nil {
		return nil, fmt.Errorf("parse fcm credentials file: %w", err)
	}
	if strings.TrimSpace(credentials.ProjectID) == "" {
		return nil, errors.New("fcm credentials file does not contain project_id")
	}

	tokenSourceCredentials, err := google.CredentialsFromJSON(
		ctx,
		rawCredentials,
		"https://www.googleapis.com/auth/firebase.messaging",
	)
	if err != nil {
		return nil, fmt.Errorf("create token source from fcm credentials: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, tokenSourceCredentials.TokenSource)
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	httpClient.Timeout = cfg.Timeout

	return &FCMSender{
		endpoint:   fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", credentials.ProjectID),
		httpClient: httpClient,
	}, nil
}

func (s *FCMSender) Send(ctx context.Context, msg Message) error {
	requestBody, err := json.Marshal(fcmSendRequest{
		Message: fcmRequestMessage{
			Token: strings.TrimSpace(msg.Token),
			Notification: &fcmNotification{
				Title: strings.TrimSpace(msg.Title),
				Body:  strings.TrimSpace(msg.Body),
			},
			Data: sanitizeData(msg.Data),
			Android: &fcmAndroidConfig{
				Priority: "high",
				Notification: &fcmAndroidNotification{
					Sound: "default",
				},
			},
			APNS: &fcmAPNSConfig{
				Headers: map[string]string{
					"apns-priority": "10",
				},
				Payload: &fcmAPNSPayload{
					APS: fcmAPS{
						Sound: "default",
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("marshal fcm request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create fcm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform fcm request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read fcm response: %w", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	return parseFCMError(resp.StatusCode, responseBody)
}

type serviceAccountCredentials struct {
	ProjectID string `json:"project_id"`
}

type fcmSendRequest struct {
	Message fcmRequestMessage `json:"message"`
}

type fcmRequestMessage struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroidConfig `json:"android,omitempty"`
	APNS         *fcmAPNSConfig    `json:"apns,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type fcmAndroidConfig struct {
	Priority     string                  `json:"priority,omitempty"`
	Notification *fcmAndroidNotification `json:"notification,omitempty"`
}

type fcmAndroidNotification struct {
	Sound string `json:"sound,omitempty"`
}

type fcmAPNSConfig struct {
	Headers map[string]string `json:"headers,omitempty"`
	Payload *fcmAPNSPayload   `json:"payload,omitempty"`
}

type fcmAPNSPayload struct {
	APS fcmAPS `json:"aps"`
}

type fcmAPS struct {
	Sound string `json:"sound,omitempty"`
}

type fcmErrorResponse struct {
	Error fcmError `json:"error"`
}

type fcmError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Status  string           `json:"status"`
	Details []fcmErrorDetail `json:"details"`
}

type fcmErrorDetail struct {
	Type      string `json:"@type"`
	ErrorCode string `json:"errorCode"`
}

func (e fcmError) hasUnregisteredToken() bool {
	for _, detail := range e.Details {
		if detail.ErrorCode == "UNREGISTERED" {
			return true
		}
	}

	return false
}

func parseFCMError(statusCode int, body []byte) error {
	var fcmErrResponse fcmErrorResponse
	if err := json.Unmarshal(body, &fcmErrResponse); err == nil {
		if fcmErrResponse.Error.hasUnregisteredToken() {
			return ErrUnregisteredToken
		}
		if fcmErrResponse.Error.Message != "" {
			return fmt.Errorf(
				"fcm send failed with status %d (%s): %s",
				statusCode,
				fcmErrResponse.Error.Status,
				fcmErrResponse.Error.Message,
			)
		}
	}

	trimmedBody := strings.TrimSpace(string(body))
	if trimmedBody == "" {
		return fmt.Errorf("fcm send failed with status %d", statusCode)
	}

	return fmt.Errorf("fcm send failed with status %d: %s", statusCode, trimmedBody)
}

func sanitizeData(data map[string]string) map[string]string {
	if len(data) == 0 {
		return nil
	}

	result := make(map[string]string, len(data))
	for key, value := range data {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil
	}

	return result
}

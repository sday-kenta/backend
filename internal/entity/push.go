package entity

import "time"

const (
	PushPlatformAndroid = "android"
	PushPlatformIOS     = "ios"

	NotificationTypeIncidentPublished = "incident_published"
	NotificationTypeIncidentRejected  = "incident_rejected"
)

type PushDevice struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	Platform   string    `json:"platform"`
	FCMToken   string    `json:"-"`
	AppVersion string    `json:"app_version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type UpsertPushDeviceInput struct {
	DeviceID   string
	Platform   string
	FCMToken   string
	AppVersion string
}

type PushNotification struct {
	RecipientUserID int64
	Type            string
	Title           string
	Body            string
	IncidentID      int64
	Status          string
	DeepLink        string
}

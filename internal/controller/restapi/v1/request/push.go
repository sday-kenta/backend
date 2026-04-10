package request

type RegisterPushDevice struct {
	DeviceID   string `json:"device_id" validate:"required,max=255" example:"0a2a97ae-9d0a-4ef1-9729-e7d8e31ab001"`
	Platform   string `json:"platform" validate:"required,oneof=android ios web pwa" example:"web"`
	FCMToken   string `json:"fcm_token" validate:"required,max=4096" example:"fcm-registration-token"`
	AppVersion string `json:"app_version,omitempty" validate:"omitempty,max=64" example:"1.0.0"`
}

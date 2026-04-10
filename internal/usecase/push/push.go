package push

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/repo"
	"github.com/sday-kenta/backend/pkg/pushclient"
)

type UseCase struct {
	repo   repo.PushDeviceRepo
	sender pushclient.Sender
}

func New(r repo.PushDeviceRepo, sender pushclient.Sender) *UseCase {
	if sender == nil {
		sender = pushclient.NewNoopSender()
	}

	return &UseCase{
		repo:   r,
		sender: sender,
	}
}

func (uc *UseCase) RegisterDevice(ctx context.Context, userID int64, input entity.UpsertPushDeviceInput) error {
	device := &entity.PushDevice{
		UserID:     userID,
		DeviceID:   strings.TrimSpace(input.DeviceID),
		Platform:   normalizePlatform(input.Platform),
		FCMToken:   strings.TrimSpace(input.FCMToken),
		AppVersion: strings.TrimSpace(input.AppVersion),
	}

	if err := uc.repo.Upsert(ctx, device); err != nil {
		return fmt.Errorf("PushUseCase - RegisterDevice - uc.repo.Upsert: %w", err)
	}

	return nil
}

func (uc *UseCase) DeleteDevice(ctx context.Context, userID int64, deviceID string) error {
	if err := uc.repo.DeleteByUserAndDeviceID(ctx, userID, strings.TrimSpace(deviceID)); err != nil {
		return fmt.Errorf("PushUseCase - DeleteDevice - uc.repo.DeleteByUserAndDeviceID: %w", err)
	}

	return nil
}

func (uc *UseCase) NotifyIncidentStatusChanged(ctx context.Context, notification entity.PushNotification) error {
	devices, err := uc.repo.ListByUserID(ctx, notification.RecipientUserID)
	if err != nil {
		return fmt.Errorf("PushUseCase - NotifyIncidentStatusChanged - uc.repo.ListByUserID: %w", err)
	}
	if len(devices) == 0 {
		return nil
	}

	payload := notificationData(notification)
	successes := 0
	var firstErr error

	for _, device := range devices {
		sendErr := uc.sender.Send(ctx, pushclient.Message{
			Token: device.FCMToken,
			Title: notification.Title,
			Body:  notification.Body,
			Data:  payload,
		})
		switch {
		case sendErr == nil:
			successes++
		case errors.Is(sendErr, pushclient.ErrUnregisteredToken):
			if deleteErr := uc.repo.DeleteByToken(ctx, device.FCMToken); deleteErr != nil && firstErr == nil {
				firstErr = fmt.Errorf("PushUseCase - NotifyIncidentStatusChanged - uc.repo.DeleteByToken: %w", deleteErr)
			}
		default:
			if firstErr == nil {
				firstErr = fmt.Errorf("PushUseCase - NotifyIncidentStatusChanged - uc.sender.Send: %w", sendErr)
			}
		}
	}

	if successes > 0 {
		return nil
	}

	return firstErr
}

func BuildIncidentStatusNotification(before, after entity.Incident, actorUserID int64) (entity.PushNotification, bool) {
	switch {
	case before.Status != entity.IncidentStatusPublished && after.Status == entity.IncidentStatusPublished:
		return entity.PushNotification{
			RecipientUserID: after.UserID,
			Type:            entity.NotificationTypeIncidentPublished,
			Title:           "Обращение опубликовано",
			Body:            fmt.Sprintf("Ваше обращение #%d опубликовано.", after.ID),
			IncidentID:      after.ID,
			Status:          after.Status,
			DeepLink:        fmt.Sprintf("/incidents/%d", after.ID),
		}, true
	case before.Status == entity.IncidentStatusReview && after.Status == entity.IncidentStatusDraft && actorUserID != after.UserID:
		return entity.PushNotification{
			RecipientUserID: after.UserID,
			Type:            entity.NotificationTypeIncidentRejected,
			Title:           "Обращение возвращено в черновик",
			Body:            fmt.Sprintf("Ваше обращение #%d возвращено администратором на доработку.", after.ID),
			IncidentID:      after.ID,
			Status:          after.Status,
			DeepLink:        fmt.Sprintf("/incidents/%d", after.ID),
		}, true
	default:
		return entity.PushNotification{}, false
	}
}

func notificationData(notification entity.PushNotification) map[string]string {
	data := map[string]string{
		"type":        notification.Type,
		"incident_id": strconv.FormatInt(notification.IncidentID, 10),
		"status":      notification.Status,
	}
	if notification.DeepLink != "" {
		data["deep_link"] = notification.DeepLink
	}

	return data
}

func normalizePlatform(platform string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "pwa" {
		return entity.PushPlatformWeb
	}

	return platform
}

package push

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/pkg/pushclient"
)

func TestBuildIncidentStatusNotificationPublished(t *testing.T) {
	t.Parallel()

	notification, ok := BuildIncidentStatusNotification(
		entity.Incident{ID: 7, UserID: 22, Status: entity.IncidentStatusReview},
		entity.Incident{ID: 7, UserID: 22, Title: "Неправильная парковка во дворе", Status: entity.IncidentStatusPublished},
		11,
	)

	require.True(t, ok)
	require.Equal(t, entity.NotificationTypeIncidentPublished, notification.Type)
	require.Equal(t, int64(22), notification.RecipientUserID)
	require.Equal(t, "Ваше обращение \"Неправильная парковка во дворе\" опубликовано.", notification.Body)
	require.Equal(t, "/incidents/7", notification.DeepLink)
}

func TestBuildIncidentStatusNotificationRejectedByAdmin(t *testing.T) {
	t.Parallel()

	notification, ok := BuildIncidentStatusNotification(
		entity.Incident{ID: 9, UserID: 33, Status: entity.IncidentStatusReview},
		entity.Incident{ID: 9, UserID: 33, Title: "Яма на дороге", Status: entity.IncidentStatusDraft},
		99,
	)

	require.True(t, ok)
	require.Equal(t, entity.NotificationTypeIncidentRejected, notification.Type)
	require.Equal(t, "Ваше обращение \"Яма на дороге\" возвращено администратором на доработку.", notification.Body)
}

func TestBuildIncidentStatusNotificationFallsBackToIncidentIDWhenTitleEmpty(t *testing.T) {
	t.Parallel()

	notification, ok := BuildIncidentStatusNotification(
		entity.Incident{ID: 11, UserID: 44, Status: entity.IncidentStatusReview},
		entity.Incident{ID: 11, UserID: 44, Title: "   ", Status: entity.IncidentStatusPublished},
		2,
	)

	require.True(t, ok)
	require.Equal(t, "Ваше обращение \"#11\" опубликовано.", notification.Body)
}

func TestBuildIncidentDeletedNotificationByAdmin(t *testing.T) {
	t.Parallel()

	notification, ok := BuildIncidentDeletedNotification(
		entity.Incident{ID: 15, UserID: 77, Title: "Сломанная скамейка", Status: entity.IncidentStatusReview},
		5,
	)

	require.True(t, ok)
	require.Equal(t, entity.NotificationTypeIncidentDeleted, notification.Type)
	require.Equal(t, int64(77), notification.RecipientUserID)
	require.Equal(t, "Ваше обращение \"Сломанная скамейка\" удалено администратором.", notification.Body)
	require.Equal(t, entity.IncidentStatusReview, notification.Status)
	require.Empty(t, notification.DeepLink)
}

func TestBuildIncidentDeletedNotificationDoesNotNotifyAuthor(t *testing.T) {
	t.Parallel()

	_, ok := BuildIncidentDeletedNotification(
		entity.Incident{ID: 15, UserID: 77, Title: "Сломанная скамейка", Status: entity.IncidentStatusReview},
		77,
	)

	require.False(t, ok)
}

func TestBuildIncidentStatusNotificationDoesNotNotifyAuthorDraft(t *testing.T) {
	t.Parallel()

	_, ok := BuildIncidentStatusNotification(
		entity.Incident{ID: 9, UserID: 33, Status: entity.IncidentStatusReview},
		entity.Incident{ID: 9, UserID: 33, Status: entity.IncidentStatusDraft},
		33,
	)

	require.False(t, ok)
}

func TestNotifyIncidentStatusChangedDeletesUnregisteredTokenAndKeepsSuccessfulSend(t *testing.T) {
	t.Parallel()

	repo := &pushDeviceRepoStub{
		devices: []entity.PushDevice{
			{FCMToken: "dead-token"},
			{FCMToken: "ok-token"},
		},
	}
	sender := &pushSenderStub{
		errs: map[string]error{
			"dead-token": pushclient.ErrUnregisteredToken,
		},
	}
	uc := New(repo, sender)

	err := uc.NotifyIncidentStatusChanged(context.Background(), entity.PushNotification{
		RecipientUserID: 1,
		Type:            entity.NotificationTypeIncidentPublished,
		Title:           "title",
		Body:            "body",
		IncidentID:      42,
		Status:          entity.IncidentStatusPublished,
		DeepLink:        "/incidents/42",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"dead-token"}, repo.deletedTokens)
	require.Len(t, sender.sent, 2)
}

type pushDeviceRepoStub struct {
	devices       []entity.PushDevice
	deletedTokens []string
	upserted      *entity.PushDevice
}

func (s *pushDeviceRepoStub) Upsert(_ context.Context, device *entity.PushDevice) error {
	deviceCopy := *device
	s.upserted = &deviceCopy
	return nil
}

func (s *pushDeviceRepoStub) ListByUserID(_ context.Context, _ int64) ([]entity.PushDevice, error) {
	return append([]entity.PushDevice(nil), s.devices...), nil
}

func (s *pushDeviceRepoStub) DeleteByUserAndDeviceID(_ context.Context, _ int64, _ string) error {
	return nil
}

func (s *pushDeviceRepoStub) DeleteByToken(_ context.Context, token string) error {
	s.deletedTokens = append(s.deletedTokens, token)
	return nil
}

type pushSenderStub struct {
	errs map[string]error
	sent []pushclient.Message
}

func (s *pushSenderStub) Send(_ context.Context, msg pushclient.Message) error {
	s.sent = append(s.sent, msg)
	if err, ok := s.errs[msg.Token]; ok {
		return err
	}

	return nil
}

func TestNotifyIncidentStatusChangedReturnsErrorWhenEverySendFails(t *testing.T) {
	t.Parallel()

	repo := &pushDeviceRepoStub{
		devices: []entity.PushDevice{
			{FCMToken: "broken-token"},
		},
	}
	sender := &pushSenderStub{
		errs: map[string]error{
			"broken-token": errors.New("boom"),
		},
	}
	uc := New(repo, sender)

	err := uc.NotifyIncidentStatusChanged(context.Background(), entity.PushNotification{
		RecipientUserID: 1,
		Type:            entity.NotificationTypeIncidentPublished,
		Title:           "title",
		Body:            "body",
		IncidentID:      42,
		Status:          entity.IncidentStatusPublished,
	})

	require.Error(t, err)
}

func TestRegisterDeviceNormalizesPWAPlatformToWeb(t *testing.T) {
	t.Parallel()

	repo := &pushDeviceRepoStub{}
	uc := New(repo, nil)

	err := uc.RegisterDevice(context.Background(), 1, entity.UpsertPushDeviceInput{
		DeviceID:   "device-1",
		Platform:   "pwa",
		FCMToken:   "token",
		AppVersion: "1.0.0",
	})

	require.NoError(t, err)
	require.NotNil(t, repo.upserted)
	require.Equal(t, entity.PushPlatformWeb, repo.upserted.Platform)
}

package pushclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFCMRequestMessageIncludesWebpushConfig(t *testing.T) {
	t.Parallel()

	msg := buildFCMRequestMessage(Message{
		Token: "token-1",
		Title: "Title",
		Body:  "Body",
		Data: map[string]string{
			"deep_link": "/incidents/42",
			"type":      "incident_published",
		},
	})

	require.Equal(t, "token-1", msg.Token)
	require.NotNil(t, msg.Webpush)
	require.Equal(t, "high", msg.Webpush.Headers["Urgency"])
	require.NotNil(t, msg.Webpush.Notification)
	require.Equal(t, "Title", msg.Webpush.Notification.Title)
	require.Equal(t, "Body", msg.Webpush.Notification.Body)
	require.Equal(t, "/incidents/42", msg.Webpush.Notification.Data["deep_link"])
	require.NotNil(t, msg.Android)
	require.NotNil(t, msg.APNS)
}

package authjwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagerGenerateAndParseToken(t *testing.T) {
	t.Parallel()

	manager := NewManager("secret", time.Hour, "backend")

	token, expiresAt, err := manager.GenerateToken(42, "admin")
	require.NoError(t, err)
	require.False(t, expiresAt.IsZero())

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, int64(42), claims.UserID)
	require.Equal(t, "admin", claims.Role)
	require.Equal(t, "backend", claims.Issuer)
}

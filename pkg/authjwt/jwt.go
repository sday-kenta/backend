package authjwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewManager(secret string, ttl time.Duration, issuer string) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
	}
}

func (m *Manager) GenerateToken(userID int64, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("authjwt - GenerateToken - SignedString: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *Manager) ParseToken(tokenString string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return Claims{}, fmt.Errorf("authjwt - ParseToken - ParseWithClaims: %w", err)
	}
	if !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	if claims.UserID == 0 {
		userID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
		if parseErr != nil || userID <= 0 {
			return Claims{}, errors.New("invalid token subject")
		}
		claims.UserID = userID
	}
	if claims.Role == "" {
		return Claims{}, errors.New("role is required in token")
	}
	if m.issuer != "" && claims.Issuer != "" && claims.Issuer != m.issuer {
		return Claims{}, errors.New("invalid token issuer")
	}

	return claims, nil
}

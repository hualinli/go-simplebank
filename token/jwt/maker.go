package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hualinli/go-simplebank/token"
)

const (
	MinSecretKeySize = 32
	TokenLeeway      = 30 * time.Second
)

type JWTMaker struct {
	secretKey string
	method    string
}

func NewJWTMaker(secretKey string) (token.Maker, error) {
	if len(secretKey) < MinSecretKeySize {
		return nil, token.ErrInvalidKeySize
	}

	return &JWTMaker{
		secretKey: secretKey,
		method:    jwt.SigningMethodHS256.Name,
	}, nil
}

func (maker *JWTMaker) CreateToken(username string, duration time.Duration) (string, *token.Payload, error) {
	now := time.Now().UTC()
	payload := &token.Payload{
		TokenID:   uuid.NewString(),
		Username:  username,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        payload.TokenID,
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.GetSigningMethod(maker.method), claims)
	signedToken, err := jwtToken.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}

	return signedToken, payload, nil
}

func (maker *JWTMaker) VerifyToken(tokenString string) (*token.Payload, error) {
	claims := &Claims{}
	jwtToken, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != maker.method {
			return nil, token.ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}, jwt.WithLeeway(TokenLeeway))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, token.ErrExpiredToken
		} else if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, token.ErrMalformedToken
		}
		return nil, token.ErrInvalidToken
	}
	if !jwtToken.Valid {
		return nil, token.ErrInvalidToken
	}
	payload := &token.Payload{
		TokenID:   claims.ID,
		Username:  claims.Username,
		IssuedAt:  claims.IssuedAt.Time.UTC(),
		ExpiredAt: claims.ExpiresAt.Time.UTC(),
	}
	return payload, nil
}

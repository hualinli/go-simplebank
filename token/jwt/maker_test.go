package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hualinli/go-simplebank/token"
	"github.com/hualinli/go-simplebank/utils"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(utils.RandomString(32))
	require.NoError(t, err)

	username := utils.RandomOwner()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	tokenString, payload, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)
	require.NotNil(t, payload)

	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)

	payload2, err := maker.VerifyToken(tokenString)
	require.NoError(t, err)
	require.NotNil(t, payload2)

	require.Equal(t, payload.TokenID, payload2.TokenID)
	require.Equal(t, username, payload2.Username)
	require.WithinDuration(t, issuedAt, payload2.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload2.ExpiredAt, time.Second)
}

func TestJWTMakerExpiredToken(t *testing.T) {
	maker, err := NewJWTMaker(utils.RandomString(32))
	require.NoError(t, err)

	tokenString, payload, err := maker.CreateToken(utils.RandomOwner(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)
	require.NotNil(t, payload)

	payload2, err := maker.VerifyToken(tokenString)
	require.ErrorIs(t, err, token.ErrExpiredToken)
	require.Nil(t, payload2)
}

func TestJWTMakerMalformedToken(t *testing.T) {
	maker, err := NewJWTMaker(utils.RandomString(32))
	require.NoError(t, err)

	payload, err := maker.VerifyToken("invalid.token.string")
	require.ErrorIs(t, err, token.ErrMalformedToken)
	require.Nil(t, payload)
}

func TestJWTMakerInvalidToken(t *testing.T) {
	username := utils.RandomOwner()
	duration := time.Minute

	verifier, err := NewJWTMaker(utils.RandomString(32))
	require.NoError(t, err)

	issuerWithDifferentKey, err := NewJWTMaker(utils.RandomString(32))
	require.NoError(t, err)

	tokenString, _, err := issuerWithDifferentKey.CreateToken(username, duration)
	require.NoError(t, err)

	payload, err := verifier.VerifyToken(tokenString)
	require.ErrorIs(t, err, token.ErrInvalidToken)
	require.Nil(t, payload)
}

func TestJWTMakerInvalidKey(t *testing.T) {
	_, err := NewJWTMaker("short")
	require.ErrorIs(t, err, token.ErrInvalidKeySize)
}

func TestJWTMakerInvalidAlgorithm(t *testing.T) {
	secretKey := utils.RandomString(32)
	maker, err := NewJWTMaker(secretKey)
	require.NoError(t, err)

	payload := &Claims{
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        utils.RandomString(10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
	tokenString, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	payload2, err := maker.VerifyToken(tokenString)
	require.ErrorIs(t, err, token.ErrInvalidToken)
	require.Nil(t, payload2)
}

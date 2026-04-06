package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/hualinli/go-simplebank/db/sqlc"
)

var (
	ErrExpiredToken = fmt.Errorf("token has expired")
)

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type refreshTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) refreshToken(ctx *gin.Context) {
	// TODO(security): add refresh token rotation with an atomic update.
	// A single refresh token should only be usable once to prevent replay under concurrent requests.
	// Suggested approach: UPDATE ... WHERE session_id=? AND refresh_token=? AND is_blocked=false AND expires_at>now() RETURNING *.

	var req refreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	payload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	payloadID, err := uuid.Parse(payload.TokenID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	now := time.Now().UTC()

	session, err := server.store.GetSession(ctx, payloadID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}

	if session.IsBlocked {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	if session.Username != payload.Username {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	if session.RefreshToken != req.RefreshToken {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	if !session.ExpiresAt.After(now) {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(payload.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	rsp := refreshTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hualinli/go-simplebank/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

var (
	ErrNotAuthorizedHeader          = fmt.Errorf("authorization header is not provided")
	ErrInvalidAuthorizationHeader   = fmt.Errorf("invalid authorization header format")
	ErrUnsupportedAuthorizationType = fmt.Errorf("unsupported authorization type")
	ErrInvalidToken                 = fmt.Errorf("invalid token")
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			// TODO: 聚合错误处理
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(ErrNotAuthorizedHeader))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(ErrInvalidAuthorizationHeader))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(ErrUnsupportedAuthorizationType))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(ErrInvalidToken))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

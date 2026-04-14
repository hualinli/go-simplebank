package gapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hualinli/go-simplebank/pb"
	"github.com/hualinli/go-simplebank/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
)

type authPayloadContextKey string

const authorizationPayloadKey authPayloadContextKey = "authorization_payload"

type authInterceptor struct {
	tokenMaker       token.Maker
	publicRPCMethods map[string]bool
}

func newAuthInterceptor(tokenMaker token.Maker) *authInterceptor {
	return &authInterceptor{
		tokenMaker: tokenMaker,
		publicRPCMethods: map[string]bool{
			pb.UserService_CreateUser_FullMethodName: true,
			pb.UserService_LoginUser_FullMethodName:  true,
		},
	}
}

func (interceptor *authInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if interceptor.publicRPCMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		payload, err := interceptor.authorize(ctx)
		if err != nil {
			return nil, toRPCError(err)
		}

		ctx = context.WithValue(ctx, authorizationPayloadKey, payload)
		return handler(ctx, req)
	}
}

func (interceptor *authInterceptor) authorize(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}

	values := md.Get(authorizationHeaderKey)
	if len(values) == 0 {
		return nil, ErrUnauthenticated
	}

	authorizationHeader := values[0]
	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		return nil, ErrUnauthenticated
	}

	authorizationType := strings.ToLower(fields[0])
	if authorizationType != authorizationTypeBearer {
		return nil, ErrUnauthenticated
	}

	accessToken := fields[1]
	payload, err := interceptor.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func authPayloadFromContext(ctx context.Context) (*token.Payload, error) {
	payload, ok := ctx.Value(authorizationPayloadKey).(*token.Payload)
	if !ok || payload == nil {
		return nil, fmt.Errorf("%w: authorization payload is missing", ErrUnauthenticated)
	}

	if strings.TrimSpace(payload.Username) == "" {
		return nil, fmt.Errorf("%w: username is missing", ErrUnauthenticated)
	}

	return payload, nil
}

func isTokenError(err error) bool {
	return errors.Is(err, token.ErrInvalidToken) ||
		errors.Is(err, token.ErrExpiredToken) ||
		errors.Is(err, token.ErrMalformedToken)
}

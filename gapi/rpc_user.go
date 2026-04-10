package gapi

import (
	"context"
	"strings"

	"github.com/google/uuid"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/pb"
	"github.com/hualinli/go-simplebank/utils"
	"google.golang.org/grpc/metadata"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req == nil {
		return nil, toRPCError(ErrInvalidRequest)
	}
	// 使用Getter方法获取字段值，避免直接访问字段可能导致的空指针异常
	err := validateCreateUserRequest(req.GetUsername(), req.GetPassword(), req.GetFullName(), req.GetEmail())
	if err != nil {
		return nil, toRPCError(err)
	}

	hashedPassword, err := utils.HashPassword(req.GetPassword())
	if err != nil {
		return nil, toRPCError(ErrInternal)
	}
	arg := db.CreateUserParams{
		Username:       req.GetUsername(),
		HashedPassword: hashedPassword,
		FullName:       req.GetFullName(),
		Email:          req.GetEmail(),
	}
	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if db.IsUniqueViolationError(err) {
			return nil, toRPCError(ErrUserExists)
		}
		return nil, toRPCError(ErrInternal)
	}
	rsp := &pb.CreateUserResponse{
		User: convertUser(&user),
	}
	return rsp, nil
}

func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	if req == nil {
		return nil, toRPCError(ErrInvalidRequest)
	}

	err := validateLoginUserRequest(req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, toRPCError(err)
	}

	user, err := server.store.GetUser(ctx, req.GetUsername())
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, toRPCError(ErrUserNotFound)
		}
		return nil, toRPCError(ErrInternal)
	}

	err = utils.CheckPassword(req.GetPassword(), user.HashedPassword)
	if err != nil {
		return nil, toRPCError(ErrInvalidPassword)
	}

	accessToken, _, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		return nil, toRPCError(ErrInternal)
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.RefreshTokenDuration)
	if err != nil {
		return nil, toRPCError(ErrInternal)
	}
	sessionID, err := uuid.Parse(refreshPayload.TokenID)
	if err != nil {
		return nil, toRPCError(ErrInternal)
	}

	userAgent, clientIP := userAgentAndClientIP(ctx)

	_, err = server.store.CreateSession(ctx, db.CreateSessionParams{
		SessionID:    sessionID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return nil, toRPCError(ErrInternal)
	}

	rsp := &pb.LoginUserResponse{
		User:         convertUser(&user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return rsp, nil
}

func userAgentAndClientIP(ctx context.Context) (string, string) {
	const defaultUA = "grpc-client"
	const defaultIP = "unknown"

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return defaultUA, defaultIP
	}

	userAgent := defaultUA
	if values := md.Get("user-agent"); len(values) > 0 && values[0] != "" {
		userAgent = values[0]
	}

	clientIP := defaultIP
	if values := md.Get("x-forwarded-for"); len(values) > 0 && values[0] != "" {
		clientIP = strings.TrimSpace(strings.Split(values[0], ",")[0])
	} else if values := md.Get("x-real-ip"); len(values) > 0 && values[0] != "" {
		clientIP = values[0]
	}

	return userAgent, clientIP
}

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
	"github.com/hualinli/go-simplebank/utils"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type userResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
} // 绝对不能在userResponse里包含HashedPassword字段

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}
	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if db.IsUniqueViolationError(err) {
			ctx.JSON(http.StatusConflict, errResponse(ErrUserOrEmailAlreadyExists))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}
	rsp := userResponse{
		Username:  user.Username,
		FullName:  user.FullName,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getUserResponse struct {
	Username string `uri:"username" binding:"required,alphanum"`
}

// 这个接口意义不大，后续可以新增一个/me接口，直接从token里获取用户名来查询用户信息，避免用户输入其他人的用户名来查询
func (server *Server) getUser(ctx *gin.Context) {
	var req getUserResponse
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if authorizationPayload.Username != req.Username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrUserNotMatch))
		return
	}
	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrUserNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}

	rsp := userResponse{
		Username:  user.Username,
		FullName:  user.FullName,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (server *Server) updateUser(ctx *gin.Context) {
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	var reqBody updateUserRequest
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	arg := db.UpdateUserParams{
		Username: username,
		FullName: reqBody.FullName,
		Email:    reqBody.Email,
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrUserNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else if db.IsUniqueViolationError(err) {
			ctx.JSON(http.StatusConflict, errResponse(ErrUserOrEmailAlreadyExists))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}

	rsp := userResponse{
		Username:  user.Username,
		FullName:  user.FullName,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, rsp)
}

type updateUserPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type updateUserPasswordResponse struct {
	Username string    `json:"username"`
	ChangeAt time.Time `json:"change_at"`
}

func (server *Server) updateUserPassword(ctx *gin.Context) {
	var req updateUserPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username

	user, err := server.store.GetUser(ctx, username)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrUserNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}

	err = utils.CheckPassword(req.OldPassword, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidPassword))
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	arg := db.UpdateUserPasswordParams{
		Username:       username,
		HashedPassword: hashedPassword,
	}

	user, err = server.store.UpdateUserPassword(ctx, arg)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrUserNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}
	changeAt := user.PasswordChangedAt.Time
	//由于不希望在unit test里依赖pgtype，所以在db层的UpdateUserPassword方法
	//里不再更新PasswordChangedAt字段，而是直接在这里根据当前时间来设置changeAt变量的值
	if changeAt.IsZero() {
		changeAt = time.Now()
	}

	rsp := updateUserPasswordResponse{
		Username: user.Username,
		ChangeAt: changeAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrUserNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}

	err = utils.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(ErrInvalidPassword))
		return
	}

	accessToken, _, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	refreshToken, payload, err := server.tokenMaker.CreateToken(user.Username, server.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	sessionID, err := uuid.Parse(payload.TokenID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	_, err = server.store.CreateSession(ctx, db.CreateSessionParams{
		SessionID:    sessionID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		ExpiresAt:    payload.ExpiredAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}
	rsp := loginUserResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: userResponse{
			Username:  user.Username,
			FullName:  user.FullName,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
		},
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) logoutUser(ctx *gin.Context) {
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	_ = authorizationPayload.Username
	// 如何实现用户登出？因为我们使用的是无状态的JWT token，所以无法在服务器端直接让某个token失效。
	// 可以考虑在数据库里维护一个token黑名单，每次请求时都检查token是否在黑名单里，如果在就拒绝请求。或者直接让客户端删除token来实现登出功能。
	ctx.JSON(http.StatusNotImplemented, errResponse(fmt.Errorf("not implemented")))
}

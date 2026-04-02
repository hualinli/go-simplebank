package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
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
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
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
		ctx.JSON(http.StatusInternalServerError, errResponse(err)) // TODO: 映射不同类型的错误到不同的HTTP状态码和错误消息
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

func (server *Server) getUser(ctx *gin.Context) {
	var req getUserResponse
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err)) // TODO: 映射不同类型的错误到不同的HTTP状态码和错误消息
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
	Username string `uri:"username" binding:"required,alphanum"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (server *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	arg := db.UpdateUserParams{
		Username: req.Username,
		FullName: req.FullName,
		Email:    req.Email,
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err)) // TODO: 映射不同类型的错误到不同的HTTP状态码和错误消息
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

type deleteUserRequest struct {
	Username string `uri:"username" binding:"required,alphanum"`
}

func (server *Server) deleteUser(ctx *gin.Context) {
	var req deleteUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	err := server.store.DeleteUser(ctx, req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err)) // TODO: 映射不同类型的错误到不同的HTTP状态码和错误消息
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

type updateUserPasswordRequest struct {
	Username    string `uri:"username" binding:"required,alphanum"`
	OldPassword string `json:"old_password" binding:"required,min=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (server *Server) updateUserPassword(ctx *gin.Context) {
	//TODO
	ctx.JSON(http.StatusNotImplemented, errResponse(fmt.Errorf("not implemented")))
}

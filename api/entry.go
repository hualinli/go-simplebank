package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
)

var (
	ErrEntryNotFound   = errors.New("entry not found")
)

type getEntryRequest struct {
	ID        int64 `uri:"id" binding:"required,min=1"`
	AccountID int64 `uri:"account" binding:"required,min=1"`
}

type getEntryResponse struct {
	ID        int64  `json:"id"`
	Amount    int64  `json:"amount"`
	CreatedAt string `json:"created_at"`
}

func (server *Server) getEntry(ctx *gin.Context) {
	var req getEntryRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	if account.Owner != username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrAccountNotMatch))
		return
	}

	entry, err := server.store.GetEntry(ctx, req.ID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrEntryNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, getEntryResponse{
		ID:        entry.ID,
		Amount:    entry.Amount,
		CreatedAt: entry.CreatedAt.Time.Format("2006-01-02 15:04:05"),
	})
}

type listEntriesRequestUri struct {
	AccountID int64 `uri:"account" binding:"required,min=1"`
}

type listEntriesRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type listEntriesResponse struct {
	Entries []getEntryResponse `json:"entries"`
}

func (server *Server) listEntries(ctx *gin.Context) {
	var reqUri listEntriesRequestUri
	if err := ctx.ShouldBindUri(&reqUri); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	account, err := server.store.GetAccount(ctx, reqUri.AccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	if account.Owner != username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrAccountNotMatch))
		return
	}
	var req listEntriesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	arg := db.ListEntriesParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	entries, err := server.store.ListEntries(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	var res listEntriesResponse
	for _, entry := range entries {
		res.Entries = append(res.Entries, getEntryResponse{
			ID:        entry.ID,
			Amount:    entry.Amount,
			CreatedAt: entry.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	ctx.JSON(http.StatusOK, res)
}

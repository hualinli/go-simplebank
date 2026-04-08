package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
)

type createTransferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

type createTransferResponse struct {
	Transfer    db.Transfer `json:"transfer"`
	FromAccount db.Account  `json:"from_account"`
	ToAccount   db.Account  `json:"to_account"`
	FromEntry   db.Entry    `json:"from_entry"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req createTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	if req.FromAccountID == req.ToAccountID {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrTransferSameAccount))
		return
	}
	fromAccount, err := server.store.GetAccount(ctx, req.FromAccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrTransferFromAccountNotFound))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		}
		return
	}
	if fromAccount.Owner != username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrTransferFromAccountNotMatch))
		return
	}
	if fromAccount.Currency != req.Currency {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrTransferCurrencyMismatch))
		return
	}

	toAccount, err := server.store.GetAccount(ctx, req.ToAccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrTransferToAccountNotFound))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		}
		return
	}
	if toAccount.Currency != req.Currency {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrTransferCurrencyMismatch))
		return
	}
	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	rsp := createTransferResponse{
		Transfer:    result.Transfer,
		FromAccount: result.FromAccount,
		ToAccount:   result.ToAccount,
		FromEntry:   result.FromEntry,
	}
	ctx.JSON(http.StatusOK, rsp)
}

type getTransferRequest struct {
	AccountID int64 `uri:"account" binding:"required,min=1"`
	ID        int64 `uri:"id" binding:"required,min=1"`
}

type getTransferResponse struct {
	Transfer db.Transfer `json:"transfer"`
}

func (server *Server) getTransfer(ctx *gin.Context) {
	var req getTransferRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	account, err := server.store.GetAccount(ctx, req.AccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		}
		return
	}
	if account.Owner != username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrAccountNotMatch))
		return
	}

	transfer, err := server.store.GetTransfer(ctx, req.ID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrTransferNotFound))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		}
		return
	}
	if transfer.FromAccountID != req.AccountID && transfer.ToAccountID != req.AccountID {
		ctx.JSON(http.StatusForbidden, errResponse(ErrTransferNotMatch))
		return
	}

	resp := getTransferResponse{
		Transfer: transfer,
	}
	ctx.JSON(http.StatusOK, resp)
}

type listTransfersRequestUri struct {
	AccountID int64 `uri:"account" binding:"required,min=1"`
}

type listTransfersRequestQuery struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

type listTransfersResponse struct {
	Transfers []db.Transfer `json:"transfers"`
}

func (server *Server) listTransfers(ctx *gin.Context) {
	var reqUri listTransfersRequestUri
	if err := ctx.ShouldBindUri(&reqUri); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	var reqQuery listTransfersRequestQuery
	if err := ctx.ShouldBindQuery(&reqQuery); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	username := authorizationPayload.Username
	account, err := server.store.GetAccount(ctx, reqUri.AccountID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		}
		return
	}
	if account.Owner != username {
		ctx.JSON(http.StatusForbidden, errResponse(ErrAccountNotMatch))
		return
	}

	arg := db.ListTransfersParams{
		FromAccountID: reqUri.AccountID,
		Limit:         reqQuery.PageSize,
		Offset:        (reqQuery.PageID - 1) * reqQuery.PageSize,
	}
	transfers, err := server.store.ListTransfers(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		return
	}

	resp := listTransfersResponse{
		Transfers: transfers,
	}
	ctx.JSON(http.StatusOK, resp)
}

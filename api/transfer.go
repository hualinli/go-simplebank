package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
)

type createTransferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req createTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	if req.FromAccountID == req.ToAccountID {
		err := fmt.Errorf("from and to account cannot be the same") // TODO: define a custom error type for this case and check with errors.Is() in the test
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	fromAccount, err := server.store.GetAccount(ctx, req.FromAccountID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errResponse(err)) // TODO: distinguish between "not found" and other errors
		return
	}
	if fromAccount.Currency != req.Currency {
		err := fmt.Errorf("from account currency mismatch: %s vs %s", fromAccount.Currency, req.Currency)
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	toAccount, err := server.store.GetAccount(ctx, req.ToAccountID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errResponse(err)) // TODO: distinguish between "not found" and other errors
		return
	}
	if toAccount.Currency != req.Currency {
		err := fmt.Errorf("to account currency mismatch: %s vs %s", toAccount.Currency, req.Currency)
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

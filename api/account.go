package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
)

type createAccountRequest struct {
	Owner    string `json:"owner" binding:"required"`
	Currency string `json:"currency" binding:"required,oneof=USD EUR CNY"`
}

type createAccountResponse struct {
	ID       int64  `json:"id"`
	Owner    string `json:"owner"`
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
}

func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	// TODO: validation
	arg := db.CreateAccountParams{
		Owner:    req.Owner,
		Currency: req.Currency,
		Balance:  0,
	}
	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	res := createAccountResponse{
		ID:       account.ID,
		Owner:    account.Owner,
		Currency: account.Currency,
		Balance:  account.Balance,
	}
	ctx.JSON(http.StatusOK, res)
}

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

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type getAccountResponse struct {
	ID       int64  `json:"id"`
	Owner    string `json:"owner"`
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
} // 预留，后期可以通过Rsp隐藏某些字段

func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// TODO: validation
	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	rsp := getAccountResponse{
		ID:       account.ID,
		Owner:    account.Owner,
		Currency: account.Currency,
		Balance:  account.Balance,
	}
	ctx.JSON(http.StatusOK, rsp)
}

type listAccountsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=10"`
}

type listAccountsResponse struct {
	Accounts []getAccountResponse `json:"accounts"`
} // 复用getAccountResponse，后期可以通过Rsp隐藏某些字段

func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	arg := db.ListAccountsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}
	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	rsp := listAccountsResponse{}
	for _, account := range accounts {
		rsp.Accounts = append(rsp.Accounts, getAccountResponse{
			ID:       account.ID,
			Owner:    account.Owner,
			Currency: account.Currency,
			Balance:  account.Balance,
		})
	}
	ctx.JSON(http.StatusOK, rsp)
}

type deleteAccountRequest struct {
	getAccountRequest
}

func (server *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	err := server.store.DeleteAccount(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "account deleted"}) // TODO: 响应内容待统一
}

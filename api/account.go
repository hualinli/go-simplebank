package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
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
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	owner := authorizationPayload.Username
	arg := db.CreateAccountParams{
		Owner:    owner,
		Currency: req.Currency,
		Balance:  0,
	}
	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		if db.IsUniqueViolationError(err) {
			ctx.JSON(http.StatusConflict, errResponse(ErrAccountAlreadyExists))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		} // owner不存在？似乎不太可能，因为owner是从token里解析出来的
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
		ctx.JSON(http.StatusBadRequest, errResponse(ErrInvalidRequest))
		return
	}

	// 用户只能看自己的账户信息
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	owner := authorizationPayload.Username

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}
	if account.Owner != owner {
		ctx.JSON(http.StatusForbidden, errResponse(ErrAccountForbidden))
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
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	owner := authorizationPayload.Username
	arg := db.ListAccountsParams{
		Owner:  owner,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}
	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
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
	authorizationPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	owner := authorizationPayload.Username
	arg := db.DeleteAccountParams{
		ID:    req.ID,
		Owner: owner,
	}
	account, err := server.store.DeleteAccount(ctx, arg)
	// 由于同时对id和owner进行查询，所以无论是账户不存在还是账户不属于用户，都会返回ErrAccountNotFound错误，因此不需要单独处理账户不属于用户的情况
	if err != nil {
		if db.IsNotFoundError(err) {
			ctx.JSON(http.StatusNotFound, errResponse(ErrAccountNotFound))
		} else if db.IsInternalError(err) {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrInternalError))
		} else if db.IsForeignKeyViolationError(err) {
			ctx.JSON(http.StatusConflict, errResponse(ErrAccountCannotBeDeleted))
			// 账户被transfer或entry引用了，无法删除
		} else {
			ctx.JSON(http.StatusInternalServerError, errResponse(ErrUnknownError))
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "account deleted", "account": account}) // TODO: 响应内容待统一
}

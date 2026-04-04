package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
	"github.com/hualinli/go-simplebank/token/jwt"
	"github.com/hualinli/go-simplebank/utils"
)

// 相当于Service层和Handler层的耦合，
// 负责处理HTTP请求，调用Store层的函数来访问数据库，并返回HTTP响应
type Server struct {
	config     utils.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

func NewServer(config utils.Config, store db.Store) (*Server, error) {
	tokenMaker, err := jwt.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create tokenmaker: %w", err)
	}
	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	u := router.Group("/users")
	{
		u.POST("", server.createUser)
		u.POST("/login", server.loginUser)

		authUsers := u.Use(authMiddleware(server.tokenMaker))
		authUsers.GET("/:username", server.getUser)
		authUsers.PUT("/:username", server.updateUser)
		authUsers.PUT("/:username/password", server.updateUserPassword)
		authUsers.POST("/logout", server.logoutUser)
	}
	a := router.Group("/accounts").Use(authMiddleware(server.tokenMaker))
	{
		a.POST("", server.createAccount)
		a.GET("/:id", server.getAccount)
		a.GET("", server.listAccounts)
		a.DELETE("/:id", server.deleteAccount)
	}
	e := router.Group("/entries").Use(authMiddleware(server.tokenMaker))
	{
		e.GET("/:id", server.getEntry)
		e.GET("", server.listEntries)
	}
	// TODO: 完善Transfers
	router.POST("/transfers", server.createTransfer)

	server.router = router
}
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

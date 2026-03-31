package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/hualinli/go-simplebank/db/sqlc"
)

// 相当于Service层和Handler层的耦合，
// 负责处理HTTP请求，调用Store层的函数来访问数据库，并返回HTTP响应
type Server struct {
	store  *db.Store
	router *gin.Engine
}

func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// 定义路由和处理函数
	router.POST("/accounts", server.createAccount)

	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

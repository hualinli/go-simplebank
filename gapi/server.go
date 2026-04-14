package gapi

import (
	"fmt"
	"net"

	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/pb"
	"github.com/hualinli/go-simplebank/token"
	"github.com/hualinli/go-simplebank/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	pb.UnimplementedUserServiceServer
	config     utils.Config
	store      db.Store
	tokenMaker token.Maker
}

func NewServer(config utils.Config, store db.Store, tokenMaker token.Maker) (*Server, error) {
	if tokenMaker == nil {
		return nil, fmt.Errorf("token maker is required")
	}

	return &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}, nil
}

func (server *Server) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("cannot listen on address %s: %w", address, err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, server)
	reflection.Register(grpcServer)
	return grpcServer.Serve(lis)
}

package main

import (
	"context"
	"log"

	"github.com/hualinli/go-simplebank/api"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/gapi"
	"github.com/hualinli/go-simplebank/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := utils.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	connPool, err := pgxpool.New(context.Background(), cfg.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer connPool.Close()

	store := db.NewStore(connPool)
	server, err := api.NewServer(cfg, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	grpcServer, err := gapi.NewServer(cfg, store, server.TokenMaker())
	if err != nil {
		log.Fatal("cannot create gRPC server:", err)
	}

	go func() {
		log.Printf("start gRPC server at %s", cfg.RPCServerAddress)
		if err := grpcServer.Start(cfg.RPCServerAddress); err != nil {
			log.Fatal("cannot start gRPC server:", err)
		}
	}()

	log.Printf("start HTTP server at %s", cfg.ServerAddress)
	err = server.Start(cfg.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}

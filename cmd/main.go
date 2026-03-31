package main

import (
	"context"
	"log"

	"github.com/hualinli/go-simplebank/api"
	db "github.com/hualinli/go-simplebank/db/sqlc"
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
	server := api.NewServer(store)

	err = server.Start(cfg.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}

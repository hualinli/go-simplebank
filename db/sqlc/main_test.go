package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/hualinli/go-simplebank/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testQueries *Queries
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	cfg, err := utils.LoadConfig("../../")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	testPool, err = pgxpool.New(context.Background(), cfg.DBSource) // 使用 = 而非 :=，赋值给全局变量
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer testPool.Close()

	testQueries = New(testPool)

	os.Exit(m.Run())
}

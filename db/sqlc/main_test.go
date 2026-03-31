package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource = "postgresql://myuser:mypassword@localhost:5432/simple_bank?sslmode=disable"
)

var testQueries *Queries
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error
	testPool, err = pgxpool.New(context.Background(), dbSource)  // 使用 = 而非 :=，赋值给全局变量
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer testPool.Close()

	testQueries = New(testPool)

	os.Exit(m.Run())
}

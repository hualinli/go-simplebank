package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

/*
由于纯Query结构体只能处理简单的单表单次查询，实际应用中经常需要处理复杂的业务逻辑，
这些逻辑可能涉及多个数据库操作，并且需要保证这些操作的原子性。
于是使用Store结构体来封装这些复杂的业务逻辑，并且在Store中使用事务来保证数据的一致性。
store有点类似DAO层，介于service和query和db之间，本质上是一个增强版的Queries，
它包含了基本的查询方法，还提供了一个execTx方法来处理事务。

store.go文件定义了Store结构体，它包含一个pgxpool.Pool连接池和一个嵌入的Queries结构体。
Store结构体提供了一个execTx方法，用于执行数据库事务。
execTx方法接受一个函数参数，该函数接受一个*Queries类型的参数，并返回一个错误。
在execTx方法中，我们首先开始一个新的数据库事务，然后创建一个新的Queries实例，并将事务传递给它。
接下来，我们调用传入的函数，并检查是否有错误发生。
如果有错误，我们尝试回滚事务，并返回相应的错误信息。
如果没有错误，我们提交事务并返回nil。
*/

// Store接口定义了所有的数据库操作，包括基本的CRUD操作和复杂的业务逻辑（如TransferTx）。
// 通过定义Store接口，我们可以将数据库操作与业务逻辑分离，便于进行Mock测试和代码维护。
type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
}

// SQLStore提供了Store接口的实现
type SQLStore struct {
	db *pgxpool.Pool
	*Queries
}

func NewStore(db *pgxpool.Pool) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// note： execTX不导出，仅公开业务函数，让service层调用，简化事务处理
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return err
	}

	q := store.WithTx(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx fn err: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx 是业务的导出函数
// 该函数首先调用 execTx 方法来执行一个事务，在事务中它会创建一个 transfer 记录，
// 然后创建两个 entry 记录，分别表示转出和转入的金额变动，最后更新两个 account 的余额。
// note：1. 并发异常：在高并发环境下，可能会出现多个事务同时修改同一账户的余额，导致数据不一致。
// 为了解决这个问题，可以使用行级锁（SELECT ... FOR UPDATE）来锁定相关的账户记录，确保同一时间只有一个事务能够修改账户余额。
// note：2. 死锁问题：加锁之后，在高并发环境下，可能会出现多个事务相互等待对方释放锁的情况，导致死锁。
// 为了避免死锁，可以确保所有事务以相同的顺序访问资源，或者使用数据库提供的死锁检测机制来自动回滚其中一个事务。
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 更新账户余额
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     arg.FromAccountID,
				Amount: -arg.Amount,
			})
			if err != nil {
				return err
			}

			result.ToAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     arg.ToAccountID,
				Amount: arg.Amount,
			})
		} else {
			result.ToAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     arg.ToAccountID,
				Amount: arg.Amount,
			})
			if err != nil {
				return err
			}

			result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
				ID:     arg.FromAccountID,
				Amount: -arg.Amount,
			})
		}
		return err
	})

	return result, err

}

package repository

import (
	"context"
	"database/sql"

	"go-wallet.in/domain"

	"github.com/doug-martin/goqu/v9"
)

type transactionRepository struct {
	db *goqu.Database
}

func NewTransaction(con *sql.DB) domain.TransactionRepository {
	return &transactionRepository{
		db: goqu.New("default", con),
	}
}

func (t *transactionRepository) Insert(ctx context.Context, transaction *domain.Transaction) error {
	dataset := t.db.Insert("transactions").Rows(goqu.Record{
		"account_id":       transaction.AccountId,
		"soft_number":      transaction.SoftNumber,
		"dof_number":       transaction.DofNumber,
		"amount":           transaction.Amount,
		"transaction_type": transaction.TransactionType,
		"transaction_date": transaction.TransactionDate,
	}).Returning("id").Executor()

	_, err := dataset.ScanStructContext(ctx, transaction)
	return err
}

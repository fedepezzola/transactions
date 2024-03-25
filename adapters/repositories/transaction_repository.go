package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/fedepezzola/transactions/business/domain"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type PostgresTransactionRepository struct {
	log *zap.SugaredLogger
	db  *sqlx.DB
}

type DBTransaction struct {
	ID                  int64     `db:"id"`
	AccountID           int64     `db:"account_id"`
	ProcessingTimestamp time.Time `db:"processing_timestamp"`
	FileTransactionID   int       `db:"file_transaction_id"`
	TrasactionMonth     int       `db:"transaction_month"`
	TrasactionDay       int       `db:"transaction_day"`
	Amount              float32   `db:"amount"`
}

func NewPostgresTransactionRepository(log *zap.SugaredLogger, db *sqlx.DB) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{
		log: log,
		db:  db,
	}
}

func (b PostgresTransactionRepository) Insert(ctx context.Context, m *domain.Transaction) (*domain.Transaction, error) {
	q := `
	INSERT INTO transactions (account_id, processing_timestamp, file_transaction_id, transaction_month, transaction_day, amount)
		 VALUES(:account_id, :processing_timestamp, :file_transaction_id, :transaction_month, :transaction_day, :amount)
		 RETURNING id;
	`

	rows, err := b.db.NamedQueryContext(ctx, q, fromTransactionDomain(m))
	if err != nil {
		return nil, fmt.Errorf("failed to insert in transactions table: %w", err)
	}
	rows.Next()
	if err = rows.Scan(&m.ID); err != nil {
		return nil, fmt.Errorf("failed to insert in accounts table: %w", err)
	}

	return m, nil
}

func fromTransactionDomain(model *domain.Transaction) *DBTransaction {
	return &DBTransaction{
		FileTransactionID:   model.FileTransactionID,
		AccountID:           model.AccountID,
		ProcessingTimestamp: model.ProcessingTimestamp,
		TrasactionMonth:     model.Month,
		TrasactionDay:       model.Day,
		Amount:              model.Amount,
	}
}

package repositories

import (
	"context"
	"fmt"

	"github.com/fedepezzola/transactions/business/domain"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type PostgresAccountRepository struct {
	log *zap.SugaredLogger
	db  *sqlx.DB
}

type DBAccount struct {
	ID            int64   `db:"id"`
	AccountNumber string  `db:"account_number"`
	Balance       float32 `db:"balance"`
}

func NewPostgresAccountRepository(log *zap.SugaredLogger, db *sqlx.DB) *PostgresAccountRepository {
	return &PostgresAccountRepository{
		log: log,
		db:  db,
	}
}

func (b PostgresAccountRepository) Insert(ctx context.Context, m *domain.Account) (*domain.Account, error) {
	q := `
	INSERT INTO accounts (account_number, balance)
		 VALUES(:account_number, :balance)
		 RETURNING id;
	`

	rows, err := b.db.NamedQueryContext(ctx, q, fromAccountDomain(m))
	if err != nil {
		return nil, fmt.Errorf("failed to insert in accounts table: %w", err)
	}
	rows.Next()
	if err = rows.Scan(&m.ID); err != nil {
		return nil, fmt.Errorf("failed to insert in accounts table: %w", err)
	}

	return m, nil
}

func (b PostgresAccountRepository) Update(ctx context.Context, m *domain.Account) (*domain.Account, error) {
	q := `
	UPDATE accounts SET
		account_number = :account_number,
		balance = :balance
		WHERE id = :id;
	`

	_, err := b.db.NamedExecContext(ctx, q, fromAccountDomain(m))
	if err != nil {
		return nil, fmt.Errorf("failed to update id %d in accounts table: %w", m.ID, err)
	}

	return m, nil
}

func (b PostgresAccountRepository) GetByAccountNumber(_ context.Context, accountNumber string) (*domain.Account, error) {
	var entities []DBAccount
	err := b.db.Select(&entities, "SELECT * FROM accounts WHERE account_number = $1 LIMIT 1", accountNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to select account_number '%s' from accounts table: %w", accountNumber, err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("entity not found")
	}
	return entities[0].toAccountDomain(), nil
}

func fromAccountDomain(model *domain.Account) *DBAccount {
	return &DBAccount{
		ID:            model.ID,
		AccountNumber: model.AccountNumber,
		Balance:       model.Balance,
	}
}

func (db DBAccount) toAccountDomain() *domain.Account {
	return &domain.Account{
		ID:            db.ID,
		AccountNumber: db.AccountNumber,
		Balance:       db.Balance,
	}
}

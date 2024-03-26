package service

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/fedepezzola/transactions/business/domain"
	"go.uber.org/zap"
)

type (
	AccountRepository interface {
		Insert(ctx context.Context, m *domain.Account) (*domain.Account, error)
		Update(ctx context.Context, m *domain.Account) (*domain.Account, error)
		GetByAccountNumber(_ context.Context, accountNumber string) (*domain.Account, error)
	}

	TransactionRepository interface {
		Insert(ctx context.Context, m *domain.Transaction) (*domain.Transaction, error)
	}

	NotificationsRepository interface {
		Notify(data interface{}) error
	}

	TransactionService struct {
		log                     *zap.SugaredLogger
		AccountRepository       AccountRepository
		TransactionRepository   TransactionRepository
		NotificationsRepository NotificationsRepository
	}

	AccountStats struct {
		Balance              float32
		FileBalance          float32
		TransactionCount     int
		TransactionsPerMonth [12]int
		DebitCount           int
		DebitAvg             float32
		CreditCount          int
		CreditAvg            float32
	}
)

func NewTransactionService(log *zap.SugaredLogger,
	accountRepository AccountRepository,
	transactionRepository TransactionRepository,
	notificationsRepository NotificationsRepository,
) *TransactionService {
	return &TransactionService{
		log:                     log,
		AccountRepository:       accountRepository,
		TransactionRepository:   transactionRepository,
		NotificationsRepository: notificationsRepository,
	}
}

func (s *TransactionService) ProcessTransactionsStream(ctx context.Context, accountNumber string, scanner *bufio.Scanner) (*AccountStats, error) {
	account, err := s.AccountRepository.GetByAccountNumber(ctx, accountNumber)
	if err != nil {
		// Assuming it fails because it does not exist
		account, err = s.AccountRepository.Insert(ctx, &domain.Account{AccountNumber: accountNumber, Balance: 0.0})
		if err != nil {
			return nil, fmt.Errorf("error retrieving account: %w", err)
		}
	}

	accountStats := AccountStats{
		Balance:              account.Balance,
		FileBalance:          0.0,
		TransactionCount:     0,
		TransactionsPerMonth: [12]int{0},
		DebitCount:           0,
		DebitAvg:             0.0,
		CreditCount:          0,
		CreditAvg:            0.0,
	}

	transaction := domain.Transaction{
		ProcessingTimestamp: time.Now(),
		AccountID:           account.ID,
	}

	for scanner.Scan() {
		fmt.Println(scanner.Text())

		txn, err := parseTransaction(scanner.Text(), &transaction)
		if err != nil {
			return nil, fmt.Errorf("error reading from file: %w", err)
		}

		txn, err = s.TransactionRepository.Insert(ctx, txn)
		if err != nil {
			return nil, fmt.Errorf("error storing transaction: %w", err)
		}

		accountStats.Balance += txn.Amount
		accountStats.FileBalance += txn.Amount
		accountStats.TransactionCount++
		accountStats.TransactionsPerMonth[txn.Month-1]++
		if txn.Amount < 0 {
			accountStats.CreditCount++
			accountStats.CreditAvg = (accountStats.CreditAvg*float32(accountStats.CreditCount-1) + txn.Amount) / float32(accountStats.CreditCount)
		} else {
			accountStats.DebitCount++
			accountStats.DebitAvg = (accountStats.DebitAvg*float32(accountStats.DebitCount-1) + txn.Amount) / float32(accountStats.DebitCount)
		}

		s.log.Info(accountStats)
	}

	account.Balance = accountStats.Balance

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from file: %w", err)
	}

	_, err = s.AccountRepository.Update(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("error updating account: %w", err)
	}

	err = s.NotificationsRepository.Notify(accountStats)
	if err != nil {
		return nil, fmt.Errorf("error notifying: %w", err)
	}

	return &accountStats, nil
}

func parseTransaction(txt string, txn *domain.Transaction) (*domain.Transaction, error) {
	p, err := fmt.Sscanf(txt, "%d,%d/%d,%f",
		&txn.FileTransactionID, &txn.Month, &txn.Day, &txn.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	if p != 4 {
		return nil, fmt.Errorf("line format error: expected data fields 4, received %d", p)
	}
	return txn, nil
}

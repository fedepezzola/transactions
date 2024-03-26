package service_test

import (
	"bufio"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fedepezzola/transactions/business/domain"
	"github.com/fedepezzola/transactions/business/service"
	"github.com/fedepezzola/transactions/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type testHelper struct {
	ctx     context.Context
	log     *zap.SugaredLogger
	service *service.TransactionService

	accountRepository       *service.MockAccountRepository
	transactionRepository   *service.MockTransactionRepository
	notificationsRepository *service.MockNotificationsRepository
}

func testSetup(t *testing.T) *testHelper {
	t.Helper()

	h := &testHelper{}

	h.ctx = context.TODO()

	h.log, _ = logger.New("TRANSACTIONS-TEST")

	h.accountRepository = &service.MockAccountRepository{}
	h.transactionRepository = &service.MockTransactionRepository{}
	h.notificationsRepository = &service.MockNotificationsRepository{}

	account := domain.Account{
		ID:            1,
		AccountNumber: "123456",
		Balance:       10.0,
	}

	h.accountRepository.EXPECT().GetByAccountNumber(h.ctx, mock.AnythingOfType("string")).Return(&account, nil)
	h.accountRepository.EXPECT().Insert(h.ctx, mock.AnythingOfType("*domain.Account")).Return(&account, nil)
	h.accountRepository.EXPECT().Update(h.ctx, mock.AnythingOfType("*domain.Account")).Return(&account, nil)

	h.notificationsRepository.EXPECT().Notify(mock.Anything).Return(nil)

	h.service = service.NewTransactionService(h.log, h.accountRepository, h.transactionRepository, h.notificationsRepository)

	return h
}

func Test_ProcessTransactionsStream_returns_stats(t *testing.T) {
	t.Parallel()
	h := testSetup(t)

	data := `0,7/15,+60.5
1,7/28,-10.3
2,8/2,-20.46
3,8/13,+10
`

	h.transactionRepository.EXPECT().Insert(h.ctx, mock.AnythingOfType("*domain.Transaction")).Return(&domain.Transaction{
		ID:                  1,
		AccountID:           1,
		ProcessingTimestamp: time.Now(),
		FileTransactionID:   0,
		Month:               7,
		Day:                 28,
		Amount:              60.5,
	}, nil).Once()
	h.transactionRepository.EXPECT().Insert(h.ctx, mock.AnythingOfType("*domain.Transaction")).Return(&domain.Transaction{
		ID:                  2,
		AccountID:           1,
		ProcessingTimestamp: time.Now(),
		FileTransactionID:   1,
		Month:               7,
		Day:                 15,
		Amount:              -10.3,
	}, nil).Once()
	h.transactionRepository.EXPECT().Insert(h.ctx, mock.AnythingOfType("*domain.Transaction")).Return(&domain.Transaction{
		ID:                  3,
		AccountID:           1,
		ProcessingTimestamp: time.Now(),
		FileTransactionID:   2,
		Month:               8,
		Day:                 2,
		Amount:              -20.46,
	}, nil).Once()
	h.transactionRepository.EXPECT().Insert(h.ctx, mock.AnythingOfType("*domain.Transaction")).Return(&domain.Transaction{
		ID:                  4,
		AccountID:           1,
		ProcessingTimestamp: time.Now(),
		FileTransactionID:   3,
		Month:               8,
		Day:                 13,
		Amount:              10,
	}, nil).Once()

	scanner := bufio.NewScanner(strings.NewReader(data))

	stats, err := h.service.ProcessTransactionsStream(h.ctx, "123456", scanner)
	assert.NoError(t, err)
	assert.Equal(t, &service.AccountStats{
		Balance:              49.74,
		FileBalance:          39.74,
		TransactionCount:     4,
		TransactionsPerMonth: [12]int{0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 0, 0},
		DebitCount:           2,
		DebitAvg:             35.25,
		CreditCount:          2,
		CreditAvg:            -15.379999,
	}, stats)
}

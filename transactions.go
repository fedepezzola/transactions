package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/fedepezzola/transactions/foundation/logger"
	"go.uber.org/zap"
)

type Transaction struct {
	Id     int
	Month  int
	Day    int
	Amount float32
}

type AccountStats struct {
	Balance              float32
	TransactionCount     int
	TransactionsPerMonth [12]int
	DebitCount           int
	DebitAvg             float32
	CreditCount          int
	CreditAvg            float32
}

func main() {
	os.Exit(mainWithExitCode())
}

func mainWithExitCode() int {
	// Construct the application logger.
	log, err := logger.New("TRANSACTIONS")
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer func(log *zap.SugaredLogger) {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}(log)

	// Perform the startup and shutdown sequence.
	if err := processFile(log); err != nil {
		log.Errorw("Fatal", "ERROR", err)
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
		return 1
	}
	return 0
}

func processFile(log *zap.SugaredLogger) error {
	file, err := os.Open("txns.csv")
	if err != nil {
		log.Errorf("Can't open file: %x", err.Error())
		return err
	}
	defer file.Close()

	accountStats := AccountStats{
		Balance:              0.0,
		TransactionCount:     0,
		TransactionsPerMonth: [12]int{0},
		DebitCount:           0,
		DebitAvg:             0.0,
		CreditCount:          0,
		CreditAvg:            0.0,
	}

	scanner := bufio.NewScanner(file)

	// discard titles form fistg line
	scanner.Scan()

	for scanner.Scan() {
		fmt.Println(scanner.Text())

		txn, err := parseTransaction(scanner.Text())
		if err != nil {
			return fmt.Errorf("error reading from file: %w", err)
		}

		accountStats.TransactionCount++
		accountStats.Balance += txn.Amount
		accountStats.TransactionsPerMonth[txn.Month-1]++
		if txn.Amount < 0 {
			accountStats.CreditCount++
			accountStats.CreditAvg = (accountStats.CreditAvg*float32(accountStats.CreditCount-1) + txn.Amount) / float32(accountStats.CreditCount)
		} else {
			accountStats.DebitCount++
			accountStats.DebitAvg = (accountStats.DebitAvg*float32(accountStats.DebitCount-1) + txn.Amount) / float32(accountStats.DebitCount)
		}

		log.Info(accountStats)
	}

	fmt.Println("Total balance is ", accountStats.Balance)
	fmt.Println("Average Debit amount: ", accountStats.DebitAvg)
	fmt.Println("Average Credit amount: ", accountStats.CreditAvg)
	for i, v := range accountStats.TransactionsPerMonth {
		if v > 0 {
			fmt.Println("Number of transactions in ", time.Month(i+1), ": ", v)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("Error reading file: %w", err.Error())
		return err
	}

	return nil
}

func parseTransaction(txt string) (*Transaction, error) {
	var txn Transaction
	p, err := fmt.Sscanf(txt, "%d,%d/%d,%f",
		&txn.Id, &txn.Month, &txn.Day, &txn.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	if p != 4 {
		return nil, fmt.Errorf("line format error: expected data fields 4, received %d", p)
	}
	return &txn, nil
}

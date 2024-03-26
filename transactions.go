package main

//go:generate go run github.com/vektra/mockery/v2@v2.42.1 --inpackage --with-expecter --all --dir ./business

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fedepezzola/transactions/adapters/repositories"
	"github.com/fedepezzola/transactions/business/service"
	"github.com/fedepezzola/transactions/infrastructure/notifications/email"

	"github.com/fedepezzola/transactions/foundation/config"
	"github.com/fedepezzola/transactions/foundation/database"
	"github.com/fedepezzola/transactions/foundation/logger"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

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

	const prefix = "TRANSACTIONS"
	cfg, help, err := config.Parse(prefix)
	if err != nil {
		fmt.Println(help)
		log.Errorf("parsing config: %w", err)
		return 1
	}

	db, err := database.Open(database.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		log.Errorw(fmt.Sprintf("failed to initialize the database: %s", err), "host", cfg.DB.Host, "name", cfg.DB.Name, "user", cfg.DB.User)
		return 1
	}
	defer func() {
		log.Infow("shutdown", "status", "stopping database support", "host", cfg.DB.Host)
		db.Close()
	}()

	file := os.Stdin

	if cfg.File != "" {
		file, err = os.Open(cfg.File)
		if err != nil {
			log.Errorf("Can't open file: %x", err.Error())
			return 1
		}
		defer file.Close()
	}

	// Perform the startup and shutdown sequence.
	if err := processFile(cfg.AccountNumber, cfg, log, db, file); err != nil {
		log.Errorw("Fatal", "ERROR", err)
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
		return 1
	}
	return 0
}

func processFile(accountNumber string, cfg config.AppConfig, log *zap.SugaredLogger, db *sqlx.DB, file *os.File) error {

	ctx := context.TODO()

	postgresAccount := repositories.NewPostgresAccountRepository(log, db)
	postgresTransaction := repositories.NewPostgresTransactionRepository(log, db)

	emailNotification := email.NewEmailNotificationListener(cfg.Notifications.Email, log)
	notificationsRepository := repositories.NewNotificationsRepository(log, []repositories.NotificationsListener{emailNotification})

	transactionService := service.NewTransactionService(log, postgresAccount, postgresTransaction, notificationsRepository)

	scanner := bufio.NewScanner(file)

	// Discard titles form first line
	scanner.Scan()

	stats, err := transactionService.ProcessTransactionsStream(ctx, accountNumber, scanner)
	if err != nil {
		return fmt.Errorf("error processing: %w", err)
	}

	fmt.Println("Total balance is ", stats.Balance)
	fmt.Println("Balance for the processed transactions is ", stats.FileBalance)
	fmt.Println("Average Debit amount: ", stats.DebitAvg)
	fmt.Println("Average Credit amount: ", stats.CreditAvg)
	for i, v := range stats.TransactionsPerMonth {
		if v > 0 {
			fmt.Println("Number of transactions in ", time.Month(i+1), ": ", v)
		}
	}

	return nil
}

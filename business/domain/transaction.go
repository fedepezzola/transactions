package domain

import "time"

type Transaction struct {
	ID                  int64
	AccountID           int64
	ProcessingTimestamp time.Time
	FileTransactionID   int
	Month               int
	Day                 int
	Amount              float32
}

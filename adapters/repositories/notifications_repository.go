package repositories

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
)

type NotificationsListener interface {
	Update(data any) error
}

type NotificationsRepository struct {
	log       *zap.SugaredLogger
	listeners []NotificationsListener
}

func NewNotificationsRepository(log *zap.SugaredLogger, listeners []NotificationsListener) *NotificationsRepository {
	return &NotificationsRepository{
		log:       log,
		listeners: listeners,
	}
}

func (n *NotificationsRepository) Notify(data any) error {
	var wrappedErrors error = nil
	for _, listener := range n.listeners {
		err := listener.Update(data)
		if err != nil {
			wrappedErrors = errors.Join(wrappedErrors, err)
		}
	}
	if wrappedErrors != nil {
		return fmt.Errorf("errors notifying: %w", wrappedErrors)
	}
	return nil
}

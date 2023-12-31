package domain

import (
	"context"
	"time"

	"go-wallet.in/dto"
)

type Notification struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Title     string    `db:"title"`
	Body      string    `db:"body"`
	Status    int8      `db:"status"`
	IsRead    int8      `db:"is_read"`
	CreatedAt time.Time `db:"created_at"`
}

type NotificationRepository interface {
	FindByUserID(ctx context.Context, userID int64) ([]Notification, error)
	Insert(ctx context.Context, notification *Notification) error
	Update(ctx context.Context, notification *Notification) error
}

type NotificationService interface {
	FindByUserID(ctx context.Context, userID int64) ([]dto.NotificationData, error)
}

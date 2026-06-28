package customermeter

import (
	"context"

	"github.com/cuffeyvidzro/leamout/internal/modules/usagecredit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	return usagecredit.NewRepository(r.db).ApplySubscriptionCredits(ctx, tx, userID, subscriptionID, checkoutID, fallbackCustomerID)
}

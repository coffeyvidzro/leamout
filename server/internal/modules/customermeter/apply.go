package customermeter

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	return r.RefreshCreditsForSubscription(ctx, tx, userID, subscriptionID, fallbackCustomerID)
}

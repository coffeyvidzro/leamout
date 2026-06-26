export type DunningAttemptStatus =
  | "pending"
  | "sent"
  | "paid"
  | "expired"
  | "canceled";

export type DunningAttemptReason = "renewal_due" | "payment_failed";

export type DunningAttempt = {
  id: string;
  user_id: string;
  subscription_id: string;
  customer_id?: string | null;
  status: DunningAttemptStatus;
  reason: DunningAttemptReason;
  period_end: string;
  expires_at: string;
  sent_at?: string | null;
  clicked_at?: string | null;
  paid_at?: string | null;
  canceled_at?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
};

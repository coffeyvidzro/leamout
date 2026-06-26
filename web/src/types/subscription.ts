export type SubscriptionStatus =
  | "active"
  | "canceled"
  | "past_due"
  | "trialing"
  | "incomplete"
  | "paused";

export type Subscription = {
  id: string;
  user_id: string;
  customer_id?: string | null;
  price_id: string;
  status: SubscriptionStatus;
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
  canceled_at?: string | null;
  ends_at?: string | null;
  ended_at?: string | null;
  customer_cancellation_reason?: string | null;
  customer_cancellation_comment?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
};

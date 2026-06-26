export type CheckoutStatus = "open" | "completed" | "expired" | "canceled";

export type CheckoutMode = "payment" | "subscription" | "renewal";

export type CheckoutSource = "api" | "checkout_link" | "dunning" | "manual";

export type CheckoutSession = {
  id: string;
  user_id: string;
  customer_id: string;
  subscription_id?: string | null;
  price_id?: string;
  mode: CheckoutMode;
  source: CheckoutSource;
  label: string;
  amount: number;
  currency: string;
  status: CheckoutStatus;
  metadata?: Record<string, unknown> | null;
  expires_at: string;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
};

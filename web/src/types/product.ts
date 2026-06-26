export type PriceType = "one_time" | "recurring" | "usage";

export type PriceInterval = "day" | "week" | "month" | "year";

export type Price = {
  id: string;
  user_id: string;
  product_id: string;
  nickname: string;
  type: PriceType;
  lookup_key?: string | null;
  unit_amount: number;
  currency: string;
  interval?: PriceInterval | null;
  metadata?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
};

export type Product = {
  id: string;
  user_id: string;
  name: string;
  description?: string | null;
  active: boolean;
  metadata?: Record<string, unknown> | null;
  prices: Price[];
  created_at: string;
  updated_at: string;
};

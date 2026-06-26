export type LedgerType = "topup" | "debit" | "refund";

export type CreditBalance = {
  user_id: string;
  balance: number;
  currency: string;
  created_at: string;
  updated_at: string;
};

export type CreditLedgerEntry = {
  id: string;
  user_id: string;
  type: LedgerType;
  amount: number;
  balance_after: number;
  destination?: string | null;
  reference?: string | null;
  description: string;
  metadata?: Record<string, unknown> | null;
  created_at: string;
};

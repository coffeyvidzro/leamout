export type Wallet = {
  id: string;
  user_id: string;
  country: string;
  currency: string;
  pending_balance: number;
  available_balance: number;
  created_at: string;
  updated_at: string;
};

export type WalletLedgerDirection = "credit" | "debit";

export type WalletBalanceType = "pending" | "available";

export type WalletLedgerReason =
  | "payment_captured"
  | "payment_settled"
  | "refund"
  | "withdrawal"
  | "adjustment";

export type WalletLedgerEntry = {
  id: string;
  wallet_id: string;
  user_id: string;
  payment_id?: string | null;
  transaction_id?: string | null;
  direction: WalletLedgerDirection;
  balance_type: WalletBalanceType;
  reason: WalletLedgerReason;
  country: string;
  currency: string;
  amount: number;
  balance_after: number;
  metadata: Record<string, unknown>;
  created_at: string;
};

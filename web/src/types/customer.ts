export type Customer = {
  id: string;
  user_id: string;
  name: string;
  email?: string | null;
  phone?: string | null;
  external_id?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
};

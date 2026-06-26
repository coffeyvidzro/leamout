import { CustomerDetail } from "@/components/customers/customer-detail";
import { serverApiFetch } from "@/lib/server-api";
import type { Customer } from "@/types/customer";
import type { DunningAttempt } from "@/types/dunning";
import type { Subscription } from "@/types/subscription";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  const [customer, subscriptions, dunningAttempts] = await Promise.all([
    serverApiFetch<Customer>(`/customers/${id}`),
    serverApiFetch<Subscription[]>("/subscriptions"),
    serverApiFetch<DunningAttempt[]>("/dunning-events"),
  ]);

  return (
    <CustomerDetail
      customer={customer}
      dunningAttempts={dunningAttempts}
      subscriptions={subscriptions}
    />
  );
}

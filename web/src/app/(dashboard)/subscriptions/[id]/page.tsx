import { SubscriptionDetail } from "@/components/subscriptions/subscription-detail";
import { serverApiFetch } from "@/lib/server-api";
import type { Customer } from "@/types/customer";
import type { DunningAttempt } from "@/types/dunning";
import type { Product } from "@/types/product";
import type { Subscription } from "@/types/subscription";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  const [subscription, customers, products, dunningAttempts] =
    await Promise.all([
      serverApiFetch<Subscription>(`/subscriptions/${id}`),
      serverApiFetch<Customer[]>("/customers"),
      serverApiFetch<Product[]>("/products"),
      serverApiFetch<DunningAttempt[]>("/dunning-events"),
    ]);

  return (
    <SubscriptionDetail
      customers={customers}
      dunningAttempts={dunningAttempts}
      products={products}
      subscription={subscription}
    />
  );
}

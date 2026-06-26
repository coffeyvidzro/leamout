import { PageHeader } from "@/components/dashboard/page-header";
import { SubscriptionCreateForm } from "@/components/subscriptions/subscription-create-form";
import { serverApiFetch } from "@/lib/server-api";
import type { Customer } from "@/types/customer";
import type { Product } from "@/types/product";

export default async function NewSubscriptionPage() {
  const [customers, products] = await Promise.all([
    serverApiFetch<Customer[]>("/customers"),
    serverApiFetch<Product[]>("/products"),
  ]);

  return (
    <div>
      <PageHeader
        title="Add subscription"
        description="Attach a customer to a product price and set the first renewal date."
      />

      <SubscriptionCreateForm customers={customers} products={products} />
    </div>
  );
}

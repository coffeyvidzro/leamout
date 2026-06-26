import { SubscriptionList } from "@/components/subscriptions/subscription-list";
import { serverApiFetch } from "@/lib/server-api";
import type { Subscription } from "@/types/subscription";

export default async function SubscriptionsPage() {
  const subscriptions = await serverApiFetch<Subscription[]>("/subscriptions");

  return <SubscriptionList subscriptions={subscriptions} />;
}

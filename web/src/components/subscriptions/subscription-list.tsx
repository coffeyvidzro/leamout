import Link from "next/link";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Subscription } from "@/types/subscription";

type SubscriptionListProps = {
  subscriptions: Subscription[];
};

function formatDate(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleDateString();
}

function getStatusClassName(status: Subscription["status"]) {
  if (status === "active") {
    return "text-emerald-600";
  }

  if (status === "past_due" || status === "incomplete") {
    return "text-amber-600";
  }

  if (status === "canceled" || status === "paused") {
    return "text-muted-foreground";
  }

  return "text-foreground";
}

function getRenewalState(subscription: Subscription) {
  if (subscription.cancel_at_period_end) {
    return "Canceling";
  }

  if (subscription.status !== "active") {
    return "Not renewing";
  }

  return "Renews";
}

export function SubscriptionList({ subscriptions }: SubscriptionListProps) {
  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Subscriptions"
          description="Track active, past due, and canceled customer subscriptions."
        />

        <Button asChild>
          <Link href="/subscriptions/new">Add subscription</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Subscription list</CardTitle>
        </CardHeader>

        <CardContent>
          {subscriptions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No subscriptions yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Price</TableHead>
                  <TableHead>Period start</TableHead>
                  <TableHead>Period end</TableHead>
                  <TableHead>Renewal</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {subscriptions.map((subscription) => {
                  const href = `/subscriptions/${subscription.id}`;

                  return (
                    <TableRow key={subscription.id}>
                      <TableCell>
                        <Button asChild className="h-auto p-0" variant="link">
                          <Link href={href}>
                            <span
                              className={`font-medium capitalize ${getStatusClassName(
                                subscription.status,
                              )}`}
                            >
                              {subscription.status.replace("_", " ")}
                            </span>
                          </Link>
                        </Button>
                      </TableCell>

                      <TableCell className="font-mono text-xs">
                        {subscription.customer_id ?? "—"}
                      </TableCell>

                      <TableCell className="font-mono text-xs">
                        {subscription.price_id}
                      </TableCell>

                      <TableCell>
                        {formatDate(subscription.current_period_start)}
                      </TableCell>

                      <TableCell>
                        {formatDate(subscription.current_period_end)}
                      </TableCell>

                      <TableCell>{getRenewalState(subscription)}</TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

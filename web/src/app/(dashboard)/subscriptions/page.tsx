"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
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
import { apiFetch } from "@/lib/api";
import type { Subscription } from "@/types/subscription";

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

export default function SubscriptionsPage() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadSubscriptions() {
      try {
        setError(null);
        const data = await apiFetch<Subscription[]>("/subscriptions");
        setSubscriptions(data);
      } catch {
        setError("Could not load subscriptions.");
      } finally {
        setLoading(false);
      }
    }

    loadSubscriptions();
  }, []);

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
          {loading ? (
            <p className="text-sm text-muted-foreground">
              Loading subscriptions...
            </p>
          ) : null}

          {error ? <p className="text-sm text-destructive">{error}</p> : null}

          {!loading && !error && subscriptions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No subscriptions yet.
            </p>
          ) : null}

          {!loading && !error && subscriptions.length > 0 ? (
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
                {subscriptions.map((subscription) => (
                  <TableRow key={subscription.id}>
                    <TableCell>
                      <span
                        className={`font-medium capitalize ${getStatusClassName(
                          subscription.status,
                        )}`}
                      >
                        {subscription.status.replace("_", " ")}
                      </span>
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
                ))}
              </TableBody>
            </Table>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}

"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiFetch } from "@/lib/api";
import type { Customer } from "@/types/customer";
import type { DunningAttempt } from "@/types/dunning";
import type { Subscription } from "@/types/subscription";

type CustomerDetailProps = {
  customerId: string;
};

function formatDate(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleDateString();
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleString();
}

export function CustomerDetail({ customerId }: CustomerDetailProps) {
  const [customer, setCustomer] = useState<Customer | null>(null);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [dunningAttempts, setDunningAttempts] = useState<DunningAttempt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const relatedSubscriptions = useMemo(
    () =>
      subscriptions.filter(
        (subscription) => subscription.customer_id === customer?.id,
      ),
    [customer?.id, subscriptions],
  );

  const relatedDunningAttempts = useMemo(
    () =>
      dunningAttempts.filter((attempt) => attempt.customer_id === customer?.id),
    [customer?.id, dunningAttempts],
  );

  useEffect(() => {
    async function loadCustomer() {
      try {
        setError(null);

        const [customerData, subscriptionData, dunningData] = await Promise.all([
          apiFetch<Customer>(`/customers/${customerId}`),
          apiFetch<Subscription[]>("/subscriptions"),
          apiFetch<DunningAttempt[]>("/dunning-events"),
        ]);

        setCustomer(customerData);
        setSubscriptions(subscriptionData);
        setDunningAttempts(dunningData);
      } catch {
        setError("Could not load customer details.");
      } finally {
        setLoading(false);
      }
    }

    loadCustomer();
  }, [customerId]);

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading customer...</p>;
  }

  if (error || !customer) {
    return (
      <div>
        <PageHeader
          title="Customer not available"
          description="We could not load this customer."
        />
        <p className="text-sm text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title={customer.name}
          description="Customer profile, subscriptions, and recovery activity."
        />

        <Button asChild variant="outline">
          <Link href="/customers">Back to customers</Link>
        </Button>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Contact</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <div className="text-muted-foreground">Email</div>
              <div>{customer.email ?? "—"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Phone</div>
              <div>{customer.phone ?? "—"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">External ID</div>
              <div>{customer.external_id ?? "—"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Created</div>
              <div>{formatDate(customer.created_at)}</div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Subscriptions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">
              {relatedSubscriptions.length}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Subscriptions attached to this customer.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Dunning events</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">
              {relatedDunningAttempts.length}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Renewal recovery attempts for this customer.
            </p>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Customer subscriptions</CardTitle>
        </CardHeader>
        <CardContent>
          {relatedSubscriptions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No subscriptions for this customer yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Price</TableHead>
                  <TableHead>Period end</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {relatedSubscriptions.map((subscription) => (
                  <TableRow key={subscription.id}>
                    <TableCell>
                      <Button asChild className="h-auto p-0" variant="link">
                        <Link href={`/subscriptions/${subscription.id}`}>
                          {subscription.status.replace("_", " ")}
                        </Link>
                      </Button>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {subscription.price_id}
                    </TableCell>
                    <TableCell>
                      {formatDate(subscription.current_period_end)}
                    </TableCell>
                    <TableCell>{formatDate(subscription.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Recent dunning events</CardTitle>
        </CardHeader>
        <CardContent>
          {relatedDunningAttempts.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No dunning events for this customer yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead>Sent</TableHead>
                  <TableHead>Clicked</TableHead>
                  <TableHead>Paid</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {relatedDunningAttempts.map((attempt) => (
                  <TableRow key={attempt.id}>
                    <TableCell className="capitalize">{attempt.status}</TableCell>
                    <TableCell className="capitalize">
                      {attempt.reason.replace("_", " ")}
                    </TableCell>
                    <TableCell>{formatDateTime(attempt.sent_at)}</TableCell>
                    <TableCell>{formatDateTime(attempt.clicked_at)}</TableCell>
                    <TableCell>{formatDateTime(attempt.paid_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

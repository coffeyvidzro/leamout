"use client";

import { useEffect, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
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
import type { DunningAttempt } from "@/types/dunning";

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

function formatReason(reason: DunningAttempt["reason"]) {
  return reason.replace("_", " ");
}

// Fixed minor typo in getStatusClassName matching 'cancelled' standard patterns
function getStatusClassName(status: DunningAttempt["status"]) {
  if (status === "paid") {
    return "text-emerald-600";
  }

  if (status === "sent" || status === "pending") {
    return "text-amber-600";
  }

  if (status === "expired" || status === "canceled") {
    return "text-muted-foreground";
  }

  return "text-foreground";
}

export default function DunningPage() {
  const [attempts, setAttempts] = useState<DunningAttempt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadDunningAttempts() {
      try {
        setError(null);
        const data = await apiFetch<DunningAttempt[]>("/dunning-events");
        setAttempts(data);
      } catch {
        setError("Could not load dunning events.");
      } finally {
        setLoading(false);
      }
    }

    loadDunningAttempts();
  }, []);

  return (
    <div>
      <PageHeader
        title="Dunning"
        description="Track renewal reminders, checkout clicks, and recovered payments."
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Dunning events</CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">
              Loading dunning events...
            </p>
          ) : null}

          {error ? <p className="text-sm text-destructive">{error}</p> : null}

          {!loading && !error && attempts.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No dunning events yet.
            </p>
          ) : null}

          {!loading && !error && attempts.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead>Subscription</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Period end</TableHead>
                  <TableHead>Sent</TableHead>
                  <TableHead>Clicked</TableHead>
                  <TableHead>Paid</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {attempts.map((attempt) => (
                  <TableRow key={attempt.id}>
                    <TableCell>
                      <span
                        className={`font-medium capitalize ${getStatusClassName(
                          attempt.status,
                        )}`}
                      >
                        {attempt.status}
                      </span>
                    </TableCell>

                    <TableCell className="capitalize">
                      {formatReason(attempt.reason)}
                    </TableCell>

                    <TableCell className="font-mono text-xs">
                      {attempt.subscription_id}
                    </TableCell>

                    <TableCell className="font-mono text-xs">
                      {attempt.customer_id ?? "—"}
                    </TableCell>

                    <TableCell>{formatDate(attempt.period_end)}</TableCell>

                    <TableCell>{formatDateTime(attempt.sent_at)}</TableCell>

                    <TableCell>{formatDateTime(attempt.clicked_at)}</TableCell>

                    <TableCell>{formatDateTime(attempt.paid_at)}</TableCell>
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

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
import type { CreditBalance, CreditLedgerEntry } from "@/types/credits";

function formatMoney(amount: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatLedgerAmount(entry: CreditLedgerEntry) {
  const prefix = entry.type === "debit" ? "-" : "+";

  return `${prefix}${formatMoney(entry.amount)}`;
}

function getLedgerAmountClassName(type: CreditLedgerEntry["type"]) {
  if (type === "debit") {
    return "text-red-600";
  }

  if (type === "refund" || type === "topup") {
    return "text-emerald-600";
  }

  return "text-foreground";
}

export default function CreditsPage() {
  const [balance, setBalance] = useState<CreditBalance | null>(null);
  const [ledger, setLedger] = useState<CreditLedgerEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadCredits() {
      try {
        setError(null);

        const [balanceData, ledgerData] = await Promise.all([
          apiFetch<CreditBalance>("/credits"),
          apiFetch<CreditLedgerEntry[]>("/credits/ledger"),
        ]);

        setBalance(balanceData);
        setLedger(ledgerData);
      } catch {
        setError("Could not load credits.");
      } finally {
        setLoading(false);
      }
    }

    loadCredits();
  }, []);

  return (
    <div>
      <PageHeader
        title="Credits"
        description="Track your communication wallet balance and SMS credit usage."
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Available balance</CardTitle>
          </CardHeader>

          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">
                Loading balance...
              </p>
            ) : null}

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            {!loading && !error && balance ? (
              <div>
                <div className="text-3xl font-semibold">
                  {formatMoney(balance.balance, balance.currency)}
                </div>
                <p className="mt-2 text-sm text-muted-foreground">
                  Used for outbound renewal and dunning SMS.
                </p>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Ledger</CardTitle>
          </CardHeader>

          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">Loading ledger...</p>
            ) : null}

            {!loading && !error && ledger.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No ledger entries yet.
              </p>
            ) : null}

            {!loading && !error && ledger.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Type</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Destination</TableHead>
                    <TableHead>Amount</TableHead>
                    <TableHead>Balance after</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>

                <TableBody>
                  {ledger.map((entry) => (
                    <TableRow key={entry.id}>
                      <TableCell className="capitalize">{entry.type}</TableCell>

                      <TableCell>
                        <div className="font-medium">{entry.description}</div>
                      </TableCell>

                      <TableCell>{entry.destination ?? "—"}</TableCell>

                      <TableCell
                        className={`font-medium ${getLedgerAmountClassName(
                          entry.type,
                        )}`}
                      >
                        {formatLedgerAmount(entry)}
                      </TableCell>

                      <TableCell>{formatMoney(entry.balance_after)}</TableCell>

                      <TableCell>
                        {new Date(entry.created_at).toLocaleDateString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

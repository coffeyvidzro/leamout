"use client";

import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
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
import type { Wallet, WalletLedgerEntry } from "@/types/wallet";

function formatMoney(amount: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatReason(reason: string) {
  return reason.replaceAll("_", " ");
}

function formatLedgerAmount(entry: WalletLedgerEntry) {
  const prefix = entry.direction === "debit" ? "-" : "+";

  return `${prefix}${formatMoney(entry.amount, entry.currency)}`;
}

function getLedgerAmountClassName(direction: WalletLedgerEntry["direction"]) {
  if (direction === "debit") {
    return "text-red-600";
  }

  return "text-emerald-600";
}

export default function WalletPage() {
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [ledger, setLedger] = useState<WalletLedgerEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const primaryWallet = useMemo(() => {
    return (
      wallets.find(
        (wallet) => wallet.country === "GH" && wallet.currency === "GHS",
      ) ?? wallets[0]
    );
  }, [wallets]);

  useEffect(() => {
    async function loadWallet() {
      try {
        setError(null);

        const [walletsData, ledgerData] = await Promise.all([
          apiFetch<{ wallets: Wallet[] }>("/wallets"),
          apiFetch<{ ledger: WalletLedgerEntry[] }>("/wallets/ledger"),
        ]);

        setWallets(walletsData.wallets);
        setLedger(ledgerData.ledger);
      } catch {
        setError("Could not load wallet.");
      } finally {
        setLoading(false);
      }
    }

    loadWallet();
  }, []);

  return (
    <div>
      <PageHeader
        title="Wallet"
        description="Track captured payment revenue and wallet ledger activity."
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Available balance</CardTitle>
          </CardHeader>

          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">
                Loading wallet...
              </p>
            ) : null}

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            {!loading && !error && primaryWallet ? (
              <div>
                <div className="text-3xl font-semibold">
                  {formatMoney(
                    primaryWallet.available_balance,
                    primaryWallet.currency,
                  )}
                </div>
                <p className="mt-2 text-sm text-muted-foreground">
                  {primaryWallet.country} · {primaryWallet.currency}
                </p>
                <div className="mt-5 rounded-md border p-3 text-sm">
                  <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Pending</span>
                    <span className="font-medium">
                      {formatMoney(
                        primaryWallet.pending_balance,
                        primaryWallet.currency,
                      )}
                    </span>
                  </div>
                </div>
              </div>
            ) : null}

            {!loading && !error && !primaryWallet ? (
              <p className="text-sm text-muted-foreground">
                No wallet balance yet. Captured payments will appear here.
              </p>
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
                No wallet ledger entries yet.
              </p>
            ) : null}

            {!loading && !error && ledger.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Reason</TableHead>
                    <TableHead>Balance</TableHead>
                    <TableHead>Market</TableHead>
                    <TableHead>Amount</TableHead>
                    <TableHead>Balance after</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>

                <TableBody>
                  {ledger.map((entry) => (
                    <TableRow key={entry.id}>
                      <TableCell className="capitalize">
                        {formatReason(entry.reason)}
                      </TableCell>

                      <TableCell className="capitalize">
                        {entry.balance_type}
                      </TableCell>

                      <TableCell>
                        {entry.country} · {entry.currency}
                      </TableCell>

                      <TableCell
                        className={`font-medium ${getLedgerAmountClassName(
                          entry.direction,
                        )}`}
                      >
                        {formatLedgerAmount(entry)}
                      </TableCell>

                      <TableCell>
                        {formatMoney(entry.balance_after, entry.currency)}
                      </TableCell>

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

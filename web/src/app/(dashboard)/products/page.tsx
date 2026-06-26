"use client";

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
import type { Product } from "@/types/product";

function formatMoney(amount: number, currency: string) {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatPrice(product: Product) {
  const price = product.prices[0];

  if (!price) {
    return "No price";
  }

  const amount = formatMoney(price.unit_amount, price.currency);

  if (price.type === "recurring" && price.interval) {
    return `${amount} / ${price.interval}`;
  }

  return amount;
}

export default function ProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadProducts() {
      try {
        setError(null);
        const data = await apiFetch<Product[]>("/products");
        setProducts(data);
      } catch {
        setError("Could not load products.");
      } finally {
        setLoading(false);
      }
    }

    loadProducts();
  }, []);

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Products"
          description="View products and prices configured for checkout."
        />

        <Button disabled>Add product</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Product list</CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading products...</p>
          ) : null}

          {error ? <p className="text-sm text-destructive">{error}</p> : null}

          {!loading && !error && products.length === 0 ? (
            <p className="text-sm text-muted-foreground">No products yet.</p>
          ) : null}

          {!loading && !error && products.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Price</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Prices</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {products.map((product) => (
                  <TableRow key={product.id}>
                    <TableCell>
                      <div className="font-medium">{product.name}</div>
                      {product.description ? (
                        <div className="text-xs text-muted-foreground">
                          {product.description}
                        </div>
                      ) : null}
                    </TableCell>

                    <TableCell>{formatPrice(product)}</TableCell>

                    <TableCell>
                      <span className="capitalize">
                        {product.active ? "active" : "inactive"}
                      </span>
                    </TableCell>

                    <TableCell>{product.prices.length}</TableCell>

                    <TableCell>
                      {new Date(product.created_at).toLocaleDateString()}
                    </TableCell>
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

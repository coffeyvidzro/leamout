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
import type { Product } from "@/types/product";

type ProductListProps = {
  products: Product[];
};

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

export function ProductList({ products }: ProductListProps) {
  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Products"
          description="View products and prices configured for checkout."
        />

        <Button asChild>
          <Link href="/products/new">Add product</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Product list</CardTitle>
        </CardHeader>

        <CardContent>
          {products.length === 0 ? (
            <p className="text-sm text-muted-foreground">No products yet.</p>
          ) : (
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
                {products.map((product) => {
                  const href = `/products/${product.id}`;

                  return (
                    <TableRow key={product.id}>
                      <TableCell>
                        <Button asChild className="h-auto p-0" variant="link">
                          <Link href={href}>{product.name}</Link>
                        </Button>
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

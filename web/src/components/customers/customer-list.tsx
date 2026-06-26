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
import type { Customer } from "@/types/customer";

type CustomerListProps = {
  customers: Customer[];
};

export function CustomerList({ customers }: CustomerListProps) {
  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Customers"
          description="View customers created through Leamout."
        />

        <Button asChild>
          <Link href="/customers/new">Add customer</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Customer list</CardTitle>
        </CardHeader>

        <CardContent>
          {customers.length === 0 ? (
            <p className="text-sm text-muted-foreground">No customers yet.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Phone</TableHead>
                  <TableHead>External ID</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {customers.map((customer) => (
                  <TableRow key={customer.id}>
                    <TableCell className="font-medium">
                      <Button asChild className="h-auto p-0" variant="link">
                        <Link href={`/customers/${customer.id}`}>
                          {customer.name}
                        </Link>
                      </Button>
                    </TableCell>
                    <TableCell>{customer.email ?? "—"}</TableCell>
                    <TableCell>{customer.phone ?? "—"}</TableCell>
                    <TableCell>{customer.external_id ?? "—"}</TableCell>
                    <TableCell>
                      {new Date(customer.created_at).toLocaleDateString()}
                    </TableCell>
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

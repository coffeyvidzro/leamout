import { CustomerList } from "@/components/customers/customer-list";
import { serverApiFetch } from "@/lib/server-api";
import type { Customer } from "@/types/customer";

export default async function CustomersPage() {
  const customers = await serverApiFetch<Customer[]>("/customers");

  return <CustomerList customers={customers} />;
}

import { CustomerCreateForm } from "@/components/customers/customer-create-form";
import { PageHeader } from "@/components/dashboard/page-header";

export default function NewCustomerPage() {
  return (
    <div>
      <PageHeader
        title="Add customer"
        description="Create a customer that can be attached to subscriptions and renewal reminders."
      />

      <CustomerCreateForm />
    </div>
  );
}

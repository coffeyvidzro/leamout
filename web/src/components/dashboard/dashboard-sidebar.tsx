import Link from "next/link";

const navItems = [
  {
    label: "Dashboard",
    href: "/dashboard",
  },
  {
    label: "Customers",
    href: "/customers",
  },
  {
    label: "Products",
    href: "/products",
  },
  {
    label: "Subscriptions",
    href: "/subscriptions",
  },
  {
    label: "Wallet",
    href: "/wallet",
  },
  {
    label: "Credits",
    href: "/credits",
  },
  {
    label: "Dunning",
    href: "/dunning",
  },
];

export function DashboardSidebar() {
  return (
    <aside className="hidden w-64 shrink-0 border-r bg-background px-4 py-6 md:block">
      <Link href="/dashboard" className="block">
        <div className="text-lg font-semibold tracking-tight">Leamout</div>
        <p className="text-xs text-muted-foreground">
          Creator billing dashboard
        </p>
      </Link>

      <nav className="mt-8 space-y-1">
        {navItems.map((item) => (
          <Link
            className="block rounded-md px-3 py-2 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground"
            href={item.href}
            key={item.href}
          >
            {item.label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}

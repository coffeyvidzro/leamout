import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-muted/30 px-6 py-12">
      <div className="mx-auto flex w-full max-w-md flex-col items-center text-center">
        <p className="text-sm font-medium text-muted-foreground">404</p>

        <h1 className="mt-3 text-3xl font-semibold tracking-tight">
          Page not found
        </h1>

        <p className="mt-3 text-sm text-muted-foreground">
          The page you are looking for does not exist or may have been moved.
        </p>

        <Button asChild className="mt-6">
          <Link href="/">Go home</Link>
        </Button>
      </div>
    </main>
  );
}

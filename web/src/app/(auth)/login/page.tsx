import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function LoginPage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-muted/30 px-6 py-12">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle>Sign in to Leamout</CardTitle>
          <CardDescription>
            Access your creator billing dashboard.
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-3">
          <Button asChild className="w-full">
            <Link href="/api/v1/auth/google">Continue with Google</Link>
          </Button>

          <Button asChild className="w-full" variant="outline">
            <Link href="/api/v1/auth/github">Continue with GitHub</Link>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}

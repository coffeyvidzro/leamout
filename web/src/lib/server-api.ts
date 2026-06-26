import { cookies } from "next/headers";

const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

export async function serverApiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const cookieStore = await cookies();
  const requestHeaders = new Headers(options?.headers);

  requestHeaders.set("Content-Type", "application/json");
  requestHeaders.set("cookie", cookieStore.toString());

  const response = await fetch(`${backendUrl}/v1${path}`, {
    ...options,
    cache: "no-store",
    headers: requestHeaders,
  });

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  return response.json() as Promise<T>;
}

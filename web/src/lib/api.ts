type APIErrorPayload = {
  error?: string;
  message?: string;
};

export class APIError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "APIError";
    this.status = status;
  }
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!response.ok) {
    throw new APIError(
      response.status,
      await errorMessageFromResponse(response),
    );
  }

  return response.json() as Promise<T>;
}

async function errorMessageFromResponse(response: Response) {
  try {
    const payload = (await response.json()) as APIErrorPayload;
    const message = payload.error ?? payload.message;
    if (typeof message === "string" && message.trim() !== "") {
      return message.trim();
    }
  } catch {
    // Fall through to the generic status message below.
  }

  return `Request failed with status ${response.status}`;
}

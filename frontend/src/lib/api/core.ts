// In-memory token storage. The token is intentionally not persisted to
// localStorage to prevent XSS from leaking a long-lived credential across
// page loads. The server sets an HttpOnly cookie on login/signup which
// handles authentication after page refreshes without any
// JavaScript-accessible persistent state.
let _token: string | null = null;

export function getToken(): string | null {
  return _token;
}

export function setToken(token: string): void {
  _token = token;
}

export function clearToken(): void {
  _token = null;
}

export function hasToken(): boolean {
  return _token !== null && _token !== "";
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function getVersion(): Promise<string> {
  const data = await request<{ version: string }>("GET", "/api/version");
  return data.version;
}

export async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  return parseResponse<T>(res);
}

// requestFormData sends a multipart/form-data request and returns the parsed
// JSON response. The caller is responsible for building the FormData body.
export async function requestFormData<T>(
  method: string,
  path: string,
  body: FormData,
): Promise<T> {
  const headers: Record<string, string> = {};

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  // Do not set Content-Type — the browser sets it automatically with the
  // correct multipart boundary when given a FormData body.
  const res = await fetch(path, { method, headers, body });

  return parseResponse<T>(res);
}

// parseResponse handles the common response parsing logic: 204 No Content,
// content-type detection, JSON parsing with fallback, and ApiError throwing.
async function parseResponse<T>(res: Response): Promise<T> {
  if (res.status === 204) {
    return undefined as T;
  }

  const contentType = res.headers.get("content-type") || "";
  const text = await res.text();
  let data: unknown;

  if (contentType.includes("application/json")) {
    if (text) {
      try {
        data = JSON.parse(text);
      } catch {
        data = { error: text };
      }
    } else {
      data = {};
    }
  } else {
    data = text ? { error: text } : {};
  }

  if (!res.ok) {
    const errorValue =
      typeof data === "object" && data !== null && "error" in data
        ? (data as { error?: unknown }).error
        : undefined;
    const message =
      typeof errorValue === "string" && errorValue.length > 0
        ? errorValue
        : res.statusText || "Request failed";
    throw new ApiError(message, res.status);
  }

  return data as T;
}

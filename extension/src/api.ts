/**
 * api.ts — Thin wrapper around the Biblioteka REST API.
 *
 * All requests use the stored API key for authentication and target the
 * server URL saved in extension storage. Every function throws on network
 * errors or non-2xx responses.
 */

import type { Settings, CaptureRequest, CaptureResponse, Library } from "../types.js";

const STORAGE_KEY = "biblioteka_settings";

/** Load settings from chrome.storage.sync. Returns partial settings when none are saved. */
export async function loadSettings(): Promise<Partial<Settings>> {
  const data = await chrome.storage.sync.get(STORAGE_KEY);
  return (data[STORAGE_KEY] as Partial<Settings>) ?? {};
}

/** Persist settings to chrome.storage.sync. */
export async function saveSettings(settings: Partial<Settings>): Promise<void> {
  await chrome.storage.sync.set({ [STORAGE_KEY]: settings });
}

/**
 * Fetch all libraries from the Biblioteka server.
 * Throws if the request fails or the server returns an error.
 */
export async function fetchLibraries(settings: Settings): Promise<Library[]> {
  const url = `${trimTrailingSlash(settings.serverURL)}/api/libraries`;
  const resp = await fetch(url, {
    method: "GET",
    headers: authHeaders(settings.apiKey),
  });
  await assertOK(resp);
  const data = await resp.json();
  // The API returns an array of library objects; normalise to the minimal shape.
  return (data as Array<{ id: string; name: string }>).map((l) => ({
    id: l.id,
    name: l.name,
  }));
}

/**
 * Submit a URL capture request to the Biblioteka server.
 * The server fetches the page, converts it to EPUB, and adds it to the library.
 * Returns the accepted response body on success; throws on failure.
 */
export async function captureURL(
  settings: Settings,
  req: CaptureRequest
): Promise<CaptureResponse> {
  const url = `${trimTrailingSlash(settings.serverURL)}/api/books/capture`;
  const resp = await fetch(url, {
    method: "POST",
    headers: {
      ...authHeaders(settings.apiKey),
      "Content-Type": "application/json",
    },
    body: JSON.stringify(req),
  });
  await assertOK(resp);
  return resp.json() as Promise<CaptureResponse>;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function authHeaders(apiKey: string): Record<string, string> {
  return { Authorization: `Bearer ${apiKey}` };
}

function trimTrailingSlash(url: string): string {
  return url.replace(/\/+$/, "");
}

async function assertOK(resp: Response): Promise<void> {
  if (resp.ok) return;
  let message = `HTTP ${resp.status}`;
  try {
    const body = (await resp.json()) as { error?: string };
    if (body.error) message = body.error;
  } catch {
    // ignore parse errors — use the status code message
  }
  throw new Error(message);
}

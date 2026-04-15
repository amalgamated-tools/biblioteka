import type { PasskeyCredential } from "../../types";
import { request, setToken } from "./core";

/**
 * Decode a base64url string to a Uint8Array.
 * The WebAuthn JSON wire format uses base64url for binary fields, but the
 * browser API expects ArrayBuffer / ArrayBufferView objects.
 */
function base64urlToBuffer(value: string): Uint8Array {
  // base64url → base64
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const pad =
    base64.length % 4 === 0 ? "" : "=".repeat(4 - (base64.length % 4));
  const binary = atob(base64 + pad);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/**
 * Convert JSON-serialized PublicKeyCredentialCreationOptions (base64url strings)
 * into the form expected by navigator.credentials.create() (ArrayBuffer fields).
 */
export function prepareCreationOptions(
  options: Record<string, unknown>,
): PublicKeyCredentialCreationOptions {
  const publicKey = (options as { publicKey: Record<string, unknown> })
    .publicKey;
  const prepared = { ...publicKey } as Record<string, unknown>;

  // challenge must be a BufferSource
  if (typeof prepared.challenge === "string") {
    prepared.challenge = base64urlToBuffer(prepared.challenge);
  }

  // user.id must be a BufferSource
  if (
    prepared.user &&
    typeof (prepared.user as Record<string, unknown>).id === "string"
  ) {
    prepared.user = {
      ...(prepared.user as Record<string, unknown>),
      id: base64urlToBuffer(
        (prepared.user as Record<string, unknown>).id as string,
      ),
    };
  }

  // excludeCredentials[].id must be BufferSource
  if (Array.isArray(prepared.excludeCredentials)) {
    prepared.excludeCredentials = (
      prepared.excludeCredentials as Record<string, unknown>[]
    ).map((c) => ({
      ...c,
      id: typeof c.id === "string" ? base64urlToBuffer(c.id) : c.id,
    }));
  }

  return prepared as unknown as PublicKeyCredentialCreationOptions;
}

/**
 * Convert JSON-serialized PublicKeyCredentialRequestOptions (base64url strings)
 * into the form expected by navigator.credentials.get() (ArrayBuffer fields).
 */
export function prepareRequestOptions(
  options: Record<string, unknown>,
): PublicKeyCredentialRequestOptions {
  const publicKey = (options as { publicKey: Record<string, unknown> })
    .publicKey;
  const prepared = { ...publicKey } as Record<string, unknown>;

  // challenge must be a BufferSource
  if (typeof prepared.challenge === "string") {
    prepared.challenge = base64urlToBuffer(prepared.challenge);
  }

  // allowCredentials[].id must be BufferSource
  if (Array.isArray(prepared.allowCredentials)) {
    prepared.allowCredentials = (
      prepared.allowCredentials as Record<string, unknown>[]
    ).map((c) => ({
      ...c,
      id: typeof c.id === "string" ? base64urlToBuffer(c.id) : c.id,
    }));
  }

  return prepared as unknown as PublicKeyCredentialRequestOptions;
}

export async function getPasskeyEnabled(): Promise<boolean> {
  const resp = await request<{ enabled: boolean }>(
    "GET",
    "/api/auth/passkey/enabled",
  );
  return resp.enabled;
}

export async function listPasskeyCredentials(): Promise<PasskeyCredential[]> {
  return request<PasskeyCredential[]>("GET", "/api/auth/passkey/credentials");
}

export async function deletePasskeyCredential(id: string): Promise<void> {
  await request<void>("DELETE", `/api/auth/passkey/credentials/${id}`);
}

export interface PasskeyBeginResponse {
  session_id: string;
  // Options are forwarded to navigator.credentials.create() / get().
  // The exact shape is defined by the WebAuthn spec and the go-webauthn library.
  options: Record<string, unknown>;
}

/**
 * Begin the passkey registration ceremony.
 * Returns the WebAuthn creation options and a server-side session ID.
 */
export async function beginPasskeyRegistration(
  name: string,
): Promise<PasskeyBeginResponse> {
  return request<PasskeyBeginResponse>(
    "POST",
    "/api/auth/passkey/register/begin",
    { name },
  );
}

/**
 * Finish the passkey registration ceremony.
 * The credential is the serialised PublicKeyCredential from navigator.credentials.create().
 */
export async function finishPasskeyRegistration(
  sessionId: string,
  credential: unknown,
): Promise<PasskeyCredential> {
  return request<PasskeyCredential>(
    "POST",
    `/api/auth/passkey/register/finish?session_id=${encodeURIComponent(sessionId)}`,
    credential,
  );
}

/**
 * Begin the passkey authentication ceremony (discoverable login).
 * Returns the WebAuthn request options and a server-side session ID.
 */
export async function beginPasskeyLogin(): Promise<PasskeyBeginResponse> {
  return request<PasskeyBeginResponse>("POST", "/api/auth/passkey/login/begin");
}

/**
 * Finish the passkey authentication ceremony.
 * The credential is the serialised PublicKeyCredential from navigator.credentials.get().
 */
export async function finishPasskeyLogin(
  sessionId: string,
  credential: unknown,
): Promise<{ token: string; user: import("../../types").User }> {
  const data = await request<{
    token: string;
    user: import("../../types").User;
  }>(
    "POST",
    `/api/auth/passkey/login/finish?session_id=${encodeURIComponent(sessionId)}`,
    credential,
  );
  setToken(data.token);
  return data;
}

import type { PasskeyCredential } from "../../types";
import { request, setToken } from "./core";

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

import type {
  ConfigStatus,
  OIDCConfig,
  SetOIDCConfigInput,
  SMTPConfig,
  SetSMTPConfigInput,
} from "../../types";
import { request } from "./core";

export async function getConfigStatus(): Promise<ConfigStatus> {
  return request<ConfigStatus>("GET", "/api/config/status");
}

export async function getOidcConfig(): Promise<OIDCConfig> {
  return request<OIDCConfig>("GET", "/api/config/oidc");
}

export async function setOidcConfig(
  config: SetOIDCConfigInput,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/oidc", config);
}

export async function createOidcLinkNonce(): Promise<string> {
  const data = await request<{ nonce: string }>(
    "POST",
    "/api/auth/oidc/link-nonce",
  );
  return data.nonce;
}

export async function getSmtpConfig(): Promise<SMTPConfig> {
  return request<SMTPConfig>("GET", "/api/config/smtp");
}

export async function setSmtpConfig(
  config: SetSMTPConfigInput,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", "/api/config/smtp", config);
}

export async function testSmtpConfig(): Promise<{ message: string }> {
  return request<{ message: string }>("POST", "/api/config/smtp/test");
}

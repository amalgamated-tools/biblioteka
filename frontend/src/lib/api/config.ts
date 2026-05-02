import type {
  ConfigStatus,
  LLMConfig,
  OIDCConfig,
  RegistrationConfig,
  SetLLMConfigInput,
  SetOIDCConfigInput,
  SMTPConfig,
  SetSMTPConfigInput,
  WatchFolderConfig,
  SetWatchFolderConfigInput,
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

export async function getWatchFolderConfig(): Promise<WatchFolderConfig> {
  return request<WatchFolderConfig>("GET", "/api/config/watch-folder");
}

export async function setWatchFolderConfig(
  config: SetWatchFolderConfigInput,
): Promise<{ message: string }> {
  return request<{ message: string }>(
    "PUT",
    "/api/config/watch-folder",
    config,
  );
}

export async function getLLMConfig(): Promise<LLMConfig> {
  return request<LLMConfig>("GET", "/api/config/llm");
}

export async function setLLMConfig(
  config: SetLLMConfigInput,
): Promise<LLMConfig> {
  return request<LLMConfig>("PUT", "/api/config/llm", config);
}

export async function getRegistrationConfig(): Promise<RegistrationConfig> {
  return request<RegistrationConfig>("GET", "/api/config/registration");
}

export async function setRegistrationConfig(
  config: RegistrationConfig,
): Promise<RegistrationConfig> {
  return request<RegistrationConfig>("PUT", "/api/config/registration", config);
}

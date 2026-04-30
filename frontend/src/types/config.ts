export interface ConfigStatus {
  oidc_configured: boolean;
  smtp_configured: boolean;
  is_admin: boolean;
}

export interface OIDCConfig {
  issuer_url: string;
  client_id: string;
  client_secret_set: boolean;
  redirect_uri: string;
}

export interface SetOIDCConfigInput {
  issuer_url: string;
  client_id: string;
  client_secret: string;
  redirect_uri: string;
}

export interface SMTPConfig {
  host: string;
  port: string;
  username: string;
  password_set: boolean;
  from: string;
  tls: string;
  env_override: boolean;
}

export interface SetSMTPConfigInput {
  host: string;
  port: string;
  username: string;
  password: string;
  from: string;
  tls: string;
}

export interface WatchFolderConfig {
  path: string;
  library_id: string;
}

export interface SetWatchFolderConfigInput {
  path: string;
  library_id: string;
}

export type LLMProvider = "" | "ollama";

export interface LLMConfig {
  provider: LLMProvider;
  endpoint: string;
  model: string;
  enabled: boolean;
  restart_required?: boolean;
}

export interface SetLLMConfigInput {
  provider: LLMProvider;
  endpoint: string;
  model: string;
  enabled: boolean;
}

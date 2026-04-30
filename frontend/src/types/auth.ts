export interface User {
  id: string;
  name: string;
  email: string;
  oidc_linked: boolean;
  is_admin: boolean;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  is_admin: boolean;
  oidc_linked: boolean;
  created_at: string;
}

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  last_used_at: string | null;
  created_at: string;
}

export interface APIKeyCreateResponse extends APIKey {
  key: string;
}

export interface KoboToken {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
}

export interface KoboTokenCreateResponse extends KoboToken {
  token: string;
}

// OPDS Credentials
// Kept separate from KosyncCredential: these are distinct protocols with
// independent backend resources. If either protocol adds protocol-specific
// fields (e.g. a KOSync device ID), the types can diverge without refactoring.

export interface OpdsCredential {
  username: string;
  created_at: string;
  updated_at: string;
}

export interface OpdsCredentialInput {
  username: string;
  password: string;
}

// KOSync Credentials

export interface KosyncCredential {
  username: string;
  created_at: string;
  updated_at: string;
}

export interface KosyncCredentialInput {
  username: string;
  password: string;
}

export interface PasskeyCredential {
  id: string;
  name: string;
  aaguid: string;
  created_at: string;
}

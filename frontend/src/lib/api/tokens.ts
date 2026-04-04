import type {
  APIKey,
  APIKeyCreateResponse,
  KoboToken,
  KoboTokenCreateResponse,
} from "../../types";
import { request } from "./core";

export async function listAPIKeys(): Promise<APIKey[]> {
  return request<APIKey[]>("GET", "/api/api-keys");
}

export async function createAPIKey(
  name: string,
): Promise<APIKeyCreateResponse> {
  return request<APIKeyCreateResponse>("POST", "/api/api-keys", { name });
}

export async function deleteAPIKey(id: string): Promise<void> {
  await request<void>("DELETE", `/api/api-keys/${id}`);
}

export async function listKoboTokens(): Promise<KoboToken[]> {
  return request<KoboToken[]>("GET", "/api/kobo/tokens");
}

export async function createKoboToken(
  name: string,
): Promise<KoboTokenCreateResponse> {
  return request<KoboTokenCreateResponse>("POST", "/api/kobo/tokens", { name });
}

export async function deleteKoboToken(id: string): Promise<void> {
  await request<void>("DELETE", `/api/kobo/tokens/${id}`);
}

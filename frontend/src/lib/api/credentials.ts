import type {
  OpdsCredential,
  OpdsCredentialInput,
  KosyncCredential,
  KosyncCredentialInput,
} from "../../types";
import { request } from "./core";

export async function getOpdsCredential(): Promise<OpdsCredential> {
  return request<OpdsCredential>("GET", "/api/opds/credentials");
}

export async function setOpdsCredential(
  input: OpdsCredentialInput,
): Promise<OpdsCredential> {
  return request<OpdsCredential>("PUT", "/api/opds/credentials", input);
}

export async function deleteOpdsCredential(): Promise<void> {
  await request<void>("DELETE", "/api/opds/credentials");
}

export async function getKosyncCredential(): Promise<KosyncCredential> {
  return request<KosyncCredential>("GET", "/api/kosync/credentials");
}

export async function setKosyncCredential(
  input: KosyncCredentialInput,
): Promise<KosyncCredential> {
  return request<KosyncCredential>("PUT", "/api/kosync/credentials", input);
}

export async function deleteKosyncCredential(): Promise<void> {
  await request<void>("DELETE", "/api/kosync/credentials");
}

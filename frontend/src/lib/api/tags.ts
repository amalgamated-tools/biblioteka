import type { Tag, TagInput } from "../../types";
import { request } from "./core";

interface TagListResponse {
  tags: Tag[];
  total: number;
  limit: number;
  offset: number;
}

export async function listTags(): Promise<Tag[]> {
  const result = await request<TagListResponse>("GET", "/api/tags");
  return result.tags;
}

export async function getTag(id: string): Promise<Tag> {
  return request<Tag>("GET", `/api/tags/${id}`);
}

export async function createTag(input: TagInput): Promise<Tag> {
  return request<Tag>("POST", "/api/tags", input);
}

export async function updateTag(id: string, input: TagInput): Promise<Tag> {
  return request<Tag>("PUT", `/api/tags/${id}`, input);
}

export async function deleteTag(id: string): Promise<void> {
  await request<void>("DELETE", `/api/tags/${id}`);
}

import type {
  GroupMemberProgress,
  ReadingGroup,
  ReadingGroupMember,
  ReadingList,
} from "../../types";
import { request } from "./core";

export async function listGroups(): Promise<ReadingGroup[]> {
  return request<ReadingGroup[]>("GET", "/api/groups");
}

export async function createGroup(
  name: string,
  description?: string | null,
): Promise<ReadingGroup> {
  return request<ReadingGroup>("POST", "/api/groups", { name, description });
}

export async function getGroup(id: string): Promise<ReadingGroup> {
  return request<ReadingGroup>("GET", `/api/groups/${id}`);
}

export async function updateGroup(
  id: string,
  name: string,
  description?: string | null,
): Promise<ReadingGroup> {
  return request<ReadingGroup>("PUT", `/api/groups/${id}`, {
    name,
    description,
  });
}

export async function deleteGroup(id: string): Promise<void> {
  await request<void>("DELETE", `/api/groups/${id}`);
}

export async function listGroupMembers(
  groupId: string,
): Promise<ReadingGroupMember[]> {
  return request<ReadingGroupMember[]>("GET", `/api/groups/${groupId}/members`);
}

export async function addGroupMember(
  groupId: string,
  userId: string,
): Promise<void> {
  await request<void>("POST", `/api/groups/${groupId}/members`, {
    user_id: userId,
  });
}

export async function removeGroupMember(
  groupId: string,
  memberId: string,
): Promise<void> {
  await request<void>("DELETE", `/api/groups/${groupId}/members/${memberId}`);
}

export async function listGroupLists(groupId: string): Promise<ReadingList[]> {
  return request<ReadingList[]>("GET", `/api/groups/${groupId}/lists`);
}

export async function addGroupList(
  groupId: string,
  listId: string,
): Promise<void> {
  await request<void>("POST", `/api/groups/${groupId}/lists`, {
    list_id: listId,
  });
}

export async function removeGroupList(
  groupId: string,
  listId: string,
): Promise<void> {
  await request<void>("DELETE", `/api/groups/${groupId}/lists/${listId}`);
}

export async function getGroupProgress(
  groupId: string,
  bookId: string,
): Promise<GroupMemberProgress[]> {
  return request<GroupMemberProgress[]>(
    "GET",
    `/api/groups/${groupId}/progress?book_id=${encodeURIComponent(bookId)}`,
  );
}

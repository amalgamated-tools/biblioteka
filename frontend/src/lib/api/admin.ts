import type { AdminUser, PaginatedAuditLogs } from "../../types";
import { request } from "./core";

export async function listUsers(): Promise<AdminUser[]> {
  return request<AdminUser[]>("GET", "/api/admin/users");
}

export async function setUserAdmin(
  userId: string,
  isAdmin: boolean,
): Promise<{ message: string }> {
  return request<{ message: string }>("PUT", `/api/admin/users/${userId}`, {
    is_admin: isAdmin,
  });
}

export async function getAuditLogs(
  limit = 50,
  offset = 0,
): Promise<PaginatedAuditLogs> {
  return request<PaginatedAuditLogs>(
    "GET",
    `/api/audit-logs?limit=${limit}&offset=${offset}`,
  );
}

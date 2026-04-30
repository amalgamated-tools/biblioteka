export interface AuditLog {
  id: string;
  /** Null for system-initiated actions (e.g. background jobs, data migrations). */
  user_id: string | null;
  action: string;
  entity_type: string;
  entity_id: string;
  /** Omitted by the backend when no metadata is present (uses omitempty). */
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface PaginatedAuditLogs {
  entries: AuditLog[];
  total: number;
  limit: number;
  offset: number;
}

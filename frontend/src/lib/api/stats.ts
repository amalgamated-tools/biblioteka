import type { MonthlyDownloads } from "../../types";
import { request } from "./core";

/**
 * Returns monthly download counts for the authenticated user.
 * @param months - Number of months to include (default 12, max 24).
 */
export async function getDownloadsPerMonth(
  months = 12,
): Promise<MonthlyDownloads[]> {
  const params = new URLSearchParams({ months: String(months) });
  return request<MonthlyDownloads[]>(
    "GET",
    `/api/stats/downloads-per-month?${params.toString()}`,
  );
}

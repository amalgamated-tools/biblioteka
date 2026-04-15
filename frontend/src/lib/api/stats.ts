import type { MonthlyDownloads, YearInBooks } from "../../types";
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

/**
 * Returns annual reading and download statistics for the authenticated user.
 * @param year - Calendar year (default: current year).
 */
export async function getYearInBooks(year?: number): Promise<YearInBooks> {
  const params = new URLSearchParams();
  if (year !== undefined) {
    params.set("year", String(year));
  }
  const query = params.toString();
  return request<YearInBooks>(
    "GET",
    `/api/stats/year-in-books${query ? `?${query}` : ""}`,
  );
}

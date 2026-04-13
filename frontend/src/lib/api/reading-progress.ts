import type { ReadingProgressStats } from "../../types";
import { request } from "./core";

export async function getReadingProgressStats(): Promise<ReadingProgressStats> {
  return request<ReadingProgressStats>("GET", "/api/reading-progress/stats");
}

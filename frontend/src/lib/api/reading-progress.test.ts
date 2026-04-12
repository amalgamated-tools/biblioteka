import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import { getReadingProgressStats, clearToken } from "../api";
import type { ReadingProgressStats } from "../../types";
import { mockFetchResponse as _mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

const fakeStats: ReadingProgressStats = {
  current_streak: 3,
  total_tracked: 5,
  total_finished: 2,
  in_progress: [
    {
      document: "my-book",
      percentage: 0.45,
      device: "Kindle",
      last_synced: "2026-04-12T10:00:00Z",
      estimated_minutes_remaining: 60,
    },
  ],
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("getReadingProgressStats", () => {
  it("calls GET /api/reading-progress/stats", async () => {
    mockFetchResponse(fakeStats);

    await getReadingProgressStats();

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/reading-progress/stats",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("returns the stats response", async () => {
    mockFetchResponse(fakeStats);

    const result = await getReadingProgressStats();

    expect(result.current_streak).toBe(3);
    expect(result.total_tracked).toBe(5);
    expect(result.total_finished).toBe(2);
    expect(result.in_progress).toHaveLength(1);
    expect(result.in_progress[0].document).toBe("my-book");
    expect(result.in_progress[0].percentage).toBe(0.45);
    expect(result.in_progress[0].device).toBe("Kindle");
    expect(result.in_progress[0].estimated_minutes_remaining).toBe(60);
  });

  it("handles empty in_progress array", async () => {
    mockFetchResponse({
      current_streak: 0,
      total_tracked: 0,
      total_finished: 0,
      in_progress: [],
    });

    const result = await getReadingProgressStats();

    expect(result.in_progress).toEqual([]);
    expect(result.current_streak).toBe(0);
  });

  it("throws on non-ok response", async () => {
    mockFetchResponse({ error: "unauthorized" }, 401);

    await expect(getReadingProgressStats()).rejects.toThrow();
  });
});

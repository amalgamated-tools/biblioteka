import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import { listUsers, setUserAdmin, getAuditLogs, clearToken } from "../api";
import type { AdminUser, PaginatedAuditLogs } from "../../types";
import {
  mockFetchResponse as _mockFetchResponse,
} from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

const fakeUser: AdminUser = {
  id: "u1",
  name: "Alice",
  email: "alice@example.com",
  is_admin: false,
  oidc_linked: false,
  created_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Admin API", () => {
  describe("listUsers", () => {
    it("sends GET /api/admin/users and returns the list", async () => {
      mockFetchResponse([fakeUser]);

      const result = await listUsers();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/admin/users");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeUser]);
    });
  });

  describe("setUserAdmin", () => {
    it("sends PUT /api/admin/users/:id with is_admin true", async () => {
      mockFetchResponse({ message: "updated" });

      const result = await setUserAdmin("u1", true);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/admin/users/u1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ is_admin: true });
      expect(result).toEqual({ message: "updated" });
    });

    it("sends PUT /api/admin/users/:id with is_admin false", async () => {
      mockFetchResponse({ message: "updated" });

      await setUserAdmin("u1", false);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/admin/users/u1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ is_admin: false });
    });
  });

  describe("getAuditLogs", () => {
    it("sends GET /api/audit-logs with default limit and offset", async () => {
      const fakeLogs: PaginatedAuditLogs = {
        entries: [],
        total: 0,
        limit: 50,
        offset: 0,
      };
      mockFetchResponse(fakeLogs);

      const result = await getAuditLogs();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/audit-logs?limit=50&offset=0");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeLogs);
    });

    it("appends custom limit and offset to the query string", async () => {
      mockFetchResponse({ entries: [], total: 100, limit: 10, offset: 20 });

      await getAuditLogs(10, 20);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/audit-logs?limit=10&offset=20");
    });
  });
});

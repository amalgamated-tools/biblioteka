import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import {
  listGroups,
  createGroup,
  getGroup,
  updateGroup,
  deleteGroup,
  listGroupMembers,
  addGroupMember,
  removeGroupMember,
  listGroupLists,
  addGroupList,
  removeGroupList,
  getGroupProgress,
  clearToken,
} from "../api";
import type {
  ReadingGroup,
  ReadingGroupMember,
  ReadingList,
  GroupMemberProgress,
} from "../../types";
import {
  mockFetchResponse as _mockFetchResponse,
  mockNoContentResponse as _mockNoContentResponse,
} from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

function mockNoContentResponse() {
  _mockNoContentResponse(fetchMock);
}

const fakeGroup: ReadingGroup = {
  id: "g-1",
  name: "Book Club",
  description: "Our club",
  owner_id: "u-1",
  member_count: 2,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeMember: ReadingGroupMember = {
  group_id: "g-1",
  user_id: "u-2",
  user_name: "alice",
  role: "member",
  joined_at: "2026-01-02T00:00:00Z",
};

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 0,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeProgress: GroupMemberProgress = {
  user_id: "u-2",
  user_name: "alice",
  percentage: 50,
  updated_at: "2026-01-03T00:00:00Z",
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Groups API", () => {
  describe("listGroups", () => {
    it("sends GET /api/groups and returns the list", async () => {
      mockFetchResponse([fakeGroup]);

      const result = await listGroups();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeGroup]);
    });
  });

  describe("createGroup", () => {
    it("sends POST /api/groups with name and returns created group", async () => {
      mockFetchResponse(fakeGroup, 201);

      const result = await createGroup("Book Club");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        name: "Book Club",
        description: undefined,
      });
      expect(result).toEqual(fakeGroup);
    });

    it("passes description when provided", async () => {
      mockFetchResponse(fakeGroup, 201);

      await createGroup("Book Club", "Our club");

      const [, options] = fetchMock.mock.calls[0];
      expect(JSON.parse(options.body)).toEqual({
        name: "Book Club",
        description: "Our club",
      });
    });
  });

  describe("getGroup", () => {
    it("sends GET /api/groups/:id and returns the group", async () => {
      mockFetchResponse(fakeGroup);

      const result = await getGroup("g-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeGroup);
    });
  });

  describe("updateGroup", () => {
    it("sends PUT /api/groups/:id with name and description", async () => {
      const updated: ReadingGroup = { ...fakeGroup, name: "Updated Club" };
      mockFetchResponse(updated);

      const result = await updateGroup("g-1", "Updated Club", "Our club");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({
        name: "Updated Club",
        description: "Our club",
      });
      expect(result).toEqual(updated);
    });

    it("sends null description to clear it", async () => {
      mockFetchResponse(fakeGroup);

      await updateGroup("g-1", "Book Club", null);

      const [, options] = fetchMock.mock.calls[0];
      expect(JSON.parse(options.body)).toEqual({
        name: "Book Club",
        description: null,
      });
    });
  });

  describe("deleteGroup", () => {
    it("sends DELETE /api/groups/:id", async () => {
      mockNoContentResponse();

      const result = await deleteGroup("g-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listGroupMembers", () => {
    it("sends GET /api/groups/:id/members and returns members", async () => {
      mockFetchResponse([fakeMember]);

      const result = await listGroupMembers("g-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/members");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeMember]);
    });
  });

  describe("addGroupMember", () => {
    it("sends POST /api/groups/:id/members with user_id", async () => {
      mockNoContentResponse();

      await addGroupMember("g-1", "u-2");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/members");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ user_id: "u-2" });
    });
  });

  describe("removeGroupMember", () => {
    it("sends DELETE /api/groups/:groupId/members/:memberUserId", async () => {
      mockNoContentResponse();

      const result = await removeGroupMember("g-1", "u-2");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/members/u-2");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("listGroupLists", () => {
    it("sends GET /api/groups/:id/lists and returns lists", async () => {
      mockFetchResponse([fakeList]);

      const result = await listGroupLists("g-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/lists");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeList]);
    });
  });

  describe("addGroupList", () => {
    it("sends POST /api/groups/:id/lists with list_id", async () => {
      mockNoContentResponse();

      await addGroupList("g-1", "rl-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/lists");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ list_id: "rl-1" });
    });
  });

  describe("removeGroupList", () => {
    it("sends DELETE /api/groups/:groupId/lists/:listId", async () => {
      mockNoContentResponse();

      const result = await removeGroupList("g-1", "rl-1");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/lists/rl-1");
      expect(options.method).toBe("DELETE");
      expect(result).toBeUndefined();
    });
  });

  describe("getGroupProgress", () => {
    it("sends GET /api/groups/:id/progress with book_id query param", async () => {
      mockFetchResponse([fakeProgress]);

      const result = await getGroupProgress("g-1", "book-42");

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/groups/g-1/progress?book_id=book-42");
      expect(options.method).toBe("GET");
      expect(result).toEqual([fakeProgress]);
    });
  });
});

import { describe, expect, it, vi, beforeEach } from "vitest";
import { createTokenManager } from "./tokenManager.svelte";

// Minimal stub items
const item1 = { id: "a", name: "Alpha" };
const item2 = { id: "b", name: "Beta" };

function makeOps(overrides: Partial<Parameters<typeof createTokenManager>[0]> = {}) {
  return {
    loadFn: vi.fn().mockResolvedValue([item1, item2]),
    deleteFn: vi.fn().mockResolvedValue(undefined),
    loadErrorMessage: "load failed",
    deleteErrorMessage: "delete failed",
    ...overrides,
  };
}

describe("createTokenManager", () => {
  describe("load", () => {
    it("populates items on success", async () => {
      const ops = makeOps();
      const mgr = createTokenManager(ops);

      expect(mgr.loading).toBe(false);
      const promise = mgr.load();
      expect(mgr.loading).toBe(true);
      await promise;

      expect(mgr.loading).toBe(false);
      expect(mgr.items).toEqual([item1, item2]);
      expect(mgr.error).toBeNull();
    });

    it("sets error on failure using the error message from the thrown Error", async () => {
      const ops = makeOps({
        loadFn: vi.fn().mockRejectedValue(new Error("network error")),
      });
      const mgr = createTokenManager(ops);

      await mgr.load();

      expect(mgr.items).toHaveLength(0);
      expect(mgr.error).toBe("network error");
      expect(mgr.loading).toBe(false);
    });

    it("falls back to loadErrorMessage for non-Error rejections", async () => {
      const ops = makeOps({
        loadFn: vi.fn().mockRejectedValue("boom"),
      });
      const mgr = createTokenManager(ops);

      await mgr.load();

      expect(mgr.error).toBe("load failed");
    });
  });

  describe("handleDelete / cancelDelete / confirmDelete", () => {
    it("sets pendingDelete when handleDelete is called", () => {
      const mgr = createTokenManager(makeOps());
      mgr.handleDelete("a", "Alpha");
      expect(mgr.pendingDelete).toEqual({ id: "a", name: "Alpha" });
    });

    it("clears pendingDelete when cancelDelete is called", async () => {
      const mgr = createTokenManager(makeOps());
      mgr.handleDelete("a", "Alpha");
      await mgr.cancelDelete();
      expect(mgr.pendingDelete).toBeNull();
    });

    it("calls deleteFn and removes item on confirmDelete", async () => {
      const ops = makeOps();
      const mgr = createTokenManager(ops);
      await mgr.load();

      mgr.handleDelete("a", "Alpha");
      await mgr.confirmDelete();

      expect(ops.deleteFn).toHaveBeenCalledWith("a");
      expect(mgr.items).toEqual([item2]);
      expect(mgr.pendingDelete).toBeNull();
    });

    it("sets error and clears pendingDelete when deleteFn rejects", async () => {
      const ops = makeOps({
        deleteFn: vi.fn().mockRejectedValue(new Error("delete error")),
      });
      const mgr = createTokenManager(ops);
      await mgr.load();

      mgr.handleDelete("a", "Alpha");
      await mgr.confirmDelete();

      expect(mgr.error).toBe("delete error");
      expect(mgr.pendingDelete).toBeNull();
      // Items should be unchanged
      expect(mgr.items).toEqual([item1, item2]);
    });

    it("does nothing when confirmDelete is called with no pending item", async () => {
      const ops = makeOps();
      const mgr = createTokenManager(ops);
      await mgr.load();

      await mgr.confirmDelete();

      expect(ops.deleteFn).not.toHaveBeenCalled();
    });
  });

  describe("setCopied / copiedId", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it("sets copiedId immediately", async () => {
      const mgr = createTokenManager(makeOps());
      await mgr.setCopied("a");
      expect(mgr.copiedId).toBe("a");
    });

    it("clears copiedId after the specified duration", async () => {
      const mgr = createTokenManager(makeOps());
      await mgr.setCopied("a", 1000);
      expect(mgr.copiedId).toBe("a");
      vi.advanceTimersByTime(1000);
      expect(mgr.copiedId).toBeNull();
    });

    it("replaces a previous copiedId when called again before timeout", async () => {
      const mgr = createTokenManager(makeOps());
      await mgr.setCopied("a", 2000);
      await mgr.setCopied("b", 2000);
      expect(mgr.copiedId).toBe("b");
    });
  });

  describe("items and error setters", () => {
    it("allows external item replacement via items setter", async () => {
      const mgr = createTokenManager(makeOps());
      await mgr.load();
      mgr.items = [item2];
      expect(mgr.items).toEqual([item2]);
    });

    it("allows external error assignment via error setter", () => {
      const mgr = createTokenManager(makeOps());
      mgr.error = "custom error";
      expect(mgr.error).toBe("custom error");
      mgr.error = null;
      expect(mgr.error).toBeNull();
    });
  });
});

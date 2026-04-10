import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { TokenListState } from "./tokenList.svelte";
import { CopyTimeoutState } from "./copyTimeout.svelte";

interface TestToken {
  id: string;
  name: string;
}

const tok1: TestToken = { id: "tok-1", name: "Token 1" };
const tok2: TestToken = { id: "tok-2", name: "Token 2" };

const loadFn = vi.fn<() => Promise<TestToken[]>>();
const deleteFn = vi.fn<(id: string) => Promise<void>>();

function makeState() {
  return new TokenListState<TestToken>({
    load: loadFn,
    delete: deleteFn,
    loadError: "Failed to load tokens",
    deleteError: "Failed to delete token",
  });
}

describe("TokenListState", () => {
  let state: TokenListState<TestToken>;

  beforeEach(() => {
    vi.clearAllMocks();
    state = makeState();
  });

  describe("load", () => {
    it("loads items and clears loading/error on success", async () => {
      loadFn.mockResolvedValue([tok1, tok2]);
      await state.load();
      expect(state.items).toEqual([tok1, tok2]);
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("sets loading to true while loading", async () => {
      let resolveFn!: (v: TestToken[]) => void;
      loadFn.mockReturnValue(
        new Promise<TestToken[]>((resolve) => {
          resolveFn = resolve;
        }),
      );

      const p = state.load();
      expect(state.loading).toBe(true);
      resolveFn([tok1]);
      await p;
      expect(state.loading).toBe(false);
    });

    it("sets error from Error instance on load failure", async () => {
      loadFn.mockRejectedValue(new Error("Network error"));
      await state.load();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("uses ops.loadError for non-Error failures", async () => {
      loadFn.mockRejectedValue("unexpected");
      await state.load();
      expect(state.error).toBe("Failed to load tokens");
    });
  });

  describe("handleDelete", () => {
    it("sets pendingDelete with the given id and name", () => {
      state.handleDelete("tok-1", "Token 1");
      expect(state.pendingDelete).toEqual({ id: "tok-1", name: "Token 1" });
    });
  });

  describe("cancelDelete", () => {
    it("clears pendingDelete", async () => {
      state.handleDelete("tok-1", "Token 1");
      await state.cancelDelete();
      expect(state.pendingDelete).toBeNull();
    });

    it("does nothing when pendingDelete is already null", async () => {
      await state.cancelDelete();
      expect(state.pendingDelete).toBeNull();
    });

    it("calls onAfterClear callback after clearing", async () => {
      const callback = vi.fn();
      state.handleDelete("tok-1", "Token 1");
      await state.cancelDelete(callback);
      expect(state.pendingDelete).toBeNull();
      expect(callback).toHaveBeenCalledOnce();
    });

    it("does not call onAfterClear when not provided", async () => {
      state.handleDelete("tok-1", "Token 1");
      await state.cancelDelete();
      expect(state.pendingDelete).toBeNull();
    });
  });

  describe("cancelDeleteWithFocus", () => {
    it("clears pendingDelete", async () => {
      state.handleDelete("tok-1", "Token 1");
      state.cancelDeleteWithFocus();
      await new Promise((r) => setTimeout(r, 0));
      expect(state.pendingDelete).toBeNull();
    });

    it("does nothing when pendingDelete is null", () => {
      state.cancelDeleteWithFocus();
      expect(state.pendingDelete).toBeNull();
    });
  });

  describe("confirmDelete", () => {
    it("does nothing when pendingDelete is null", async () => {
      await state.confirmDelete();
      expect(deleteFn).not.toHaveBeenCalled();
    });

    it("deletes the item and removes it from items", async () => {
      state.items = [tok1, tok2];
      deleteFn.mockResolvedValue(undefined);
      state.handleDelete("tok-1", "Token 1");
      await state.confirmDelete();
      expect(deleteFn).toHaveBeenCalledWith("tok-1");
      expect(state.items).toEqual([tok2]);
      expect(state.pendingDelete).toBeNull();
    });

    it("clears error before deleting", async () => {
      state.items = [tok1];
      state.error = "previous error";
      deleteFn.mockResolvedValue(undefined);
      state.handleDelete("tok-1", "Token 1");
      await state.confirmDelete();
      expect(state.error).toBeNull();
    });

    it("sets error from Error instance on delete failure", async () => {
      state.items = [tok1];
      deleteFn.mockRejectedValue(new Error("Delete failed"));
      state.handleDelete("tok-1", "Token 1");
      await state.confirmDelete();
      expect(state.error).toBe("Delete failed");
    });

    it("uses ops.deleteError for non-Error failures", async () => {
      state.items = [tok1];
      deleteFn.mockRejectedValue("unexpected");
      state.handleDelete("tok-1", "Token 1");
      await state.confirmDelete();
      expect(state.error).toBe("Failed to delete token");
    });

    it("leaves items unchanged on delete failure", async () => {
      state.items = [tok1, tok2];
      deleteFn.mockRejectedValue(new Error("fail"));
      state.handleDelete("tok-1", "Token 1");
      await state.confirmDelete();
      expect(state.items).toEqual([tok1, tok2]);
    });
  });

  describe("copy", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("exposes a CopyTimeoutState instance", () => {
      expect(state.copy).toBeInstanceOf(CopyTimeoutState);
    });

    it("copy.copiedId starts as null", () => {
      expect(state.copy.copiedId).toBeNull();
    });

    it("copy.set() updates copiedId and resets after duration", () => {
      state.copy.set("tok-1");
      expect(state.copy.copiedId).toBe("tok-1");
      vi.advanceTimersByTime(2000);
      expect(state.copy.copiedId).toBeNull();
    });

    it("copy.clear() cancels the timeout and resets copiedId", () => {
      state.copy.set("tok-1");
      state.copy.clear();
      expect(state.copy.copiedId).toBeNull();
      vi.advanceTimersByTime(2000);
      expect(state.copy.copiedId).toBeNull();
    });
  });
});

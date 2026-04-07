import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { TimeoutState } from "./timeoutState.svelte";

/** Thin subclass that exposes the protected `activate` method for testing. */
class TestTimeoutState<T> extends TimeoutState<T> {
  trigger(v: T): void {
    this.activate(v);
  }
}

describe("TimeoutState", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("with a string | null type", () => {
    it("initializes value to idleValue", () => {
      const state = new TestTimeoutState<string | null>(null, 2000);
      expect(state.value).toBeNull();
    });

    it("activate() sets value immediately", () => {
      const state = new TestTimeoutState<string | null>(null, 2000);
      state.trigger("hello");
      expect(state.value).toBe("hello");
    });

    it("reverts value to idleValue after duration", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      state.trigger("hello");
      vi.advanceTimersByTime(1000);
      expect(state.value).toBeNull();
    });

    it("does not revert before duration has elapsed", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      state.trigger("hello");
      vi.advanceTimersByTime(999);
      expect(state.value).toBe("hello");
    });

    it("cancels the previous timeout when activate() is called again", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      state.trigger("first");
      vi.advanceTimersByTime(500);
      state.trigger("second");
      expect(state.value).toBe("second");
      // Original timer would have fired at 1000 ms; advance past it
      vi.advanceTimersByTime(500);
      // New timer started at 500 ms; fires at 1500 ms total
      expect(state.value).toBe("second");
      vi.advanceTimersByTime(500);
      expect(state.value).toBeNull();
    });
  });

  describe("with a boolean type", () => {
    it("initializes value to idleValue (false)", () => {
      const state = new TestTimeoutState<boolean>(false, 3000);
      expect(state.value).toBe(false);
    });

    it("activate() sets value to true immediately", () => {
      const state = new TestTimeoutState<boolean>(false, 3000);
      state.trigger(true);
      expect(state.value).toBe(true);
    });

    it("reverts value to false after duration", () => {
      const state = new TestTimeoutState<boolean>(false, 500);
      state.trigger(true);
      vi.advanceTimersByTime(500);
      expect(state.value).toBe(false);
    });
  });

  describe("clear()", () => {
    it("resets value to idleValue immediately", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      state.trigger("hello");
      state.clear();
      expect(state.value).toBeNull();
    });

    it("cancels the pending timeout so value stays at idleValue", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      state.trigger("hello");
      state.clear();
      vi.advanceTimersByTime(2000);
      expect(state.value).toBeNull();
    });

    it("is safe to call when no timeout is pending", () => {
      const state = new TestTimeoutState<string | null>(null, 1000);
      expect(() => state.clear()).not.toThrow();
      expect(state.value).toBeNull();
    });

    it("resets a boolean state to its idleValue", () => {
      const state = new TestTimeoutState<boolean>(false, 1000);
      state.trigger(true);
      state.clear();
      expect(state.value).toBe(false);
    });
  });
});

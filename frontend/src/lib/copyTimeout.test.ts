import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { CopyTimeoutState } from "./copyTimeout.svelte";

describe("CopyTimeoutState", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("initialises with copiedId null", () => {
    const state = new CopyTimeoutState();
    expect(state.copiedId).toBeNull();
  });

  it("set() updates copiedId immediately", () => {
    const state = new CopyTimeoutState();
    state.set("tok-1");
    expect(state.copiedId).toBe("tok-1");
  });

  it("resets copiedId to null after the default 2000 ms", () => {
    const state = new CopyTimeoutState();
    state.set("tok-1");
    vi.advanceTimersByTime(2000);
    expect(state.copiedId).toBeNull();
  });

  it("respects a custom duration", () => {
    const state = new CopyTimeoutState(500);
    state.set("tok-1");
    vi.advanceTimersByTime(499);
    expect(state.copiedId).toBe("tok-1");
    vi.advanceTimersByTime(1);
    expect(state.copiedId).toBeNull();
  });

  it("cancels the previous timeout when set() is called again", () => {
    const state = new CopyTimeoutState(1000);
    state.set("tok-1");
    vi.advanceTimersByTime(500);
    state.set("tok-2");
    expect(state.copiedId).toBe("tok-2");
    // The original timer would have fired at 1000 ms total; advance past it
    vi.advanceTimersByTime(500);
    // The new timer started at 500 ms; 1000 ms from now is 1500 ms total
    expect(state.copiedId).toBe("tok-2");
    vi.advanceTimersByTime(500);
    expect(state.copiedId).toBeNull();
  });

  it("clear() resets copiedId immediately", () => {
    const state = new CopyTimeoutState(1000);
    state.set("tok-1");
    state.clear();
    expect(state.copiedId).toBeNull();
  });

  it("clear() cancels the pending timeout so copiedId stays null", () => {
    const state = new CopyTimeoutState(1000);
    state.set("tok-1");
    state.clear();
    vi.advanceTimersByTime(2000);
    expect(state.copiedId).toBeNull();
  });

  it("clear() is safe to call when no timeout is pending", () => {
    const state = new CopyTimeoutState();
    expect(() => state.clear()).not.toThrow();
    expect(state.copiedId).toBeNull();
  });
});

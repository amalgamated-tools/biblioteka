import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { SuccessTimerState } from "./successTimer.svelte";

describe("SuccessTimerState", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("initialises with visible false", () => {
    const timer = new SuccessTimerState();
    expect(timer.visible).toBe(false);
  });

  it("show() sets visible to true immediately", () => {
    const timer = new SuccessTimerState();
    timer.show();
    expect(timer.visible).toBe(true);
  });

  it("resets visible to false after the default 3000 ms", () => {
    const timer = new SuccessTimerState();
    timer.show();
    vi.advanceTimersByTime(3000);
    expect(timer.visible).toBe(false);
  });

  it("respects a custom duration", () => {
    const timer = new SuccessTimerState(500);
    timer.show();
    vi.advanceTimersByTime(499);
    expect(timer.visible).toBe(true);
    vi.advanceTimersByTime(1);
    expect(timer.visible).toBe(false);
  });

  it("cancels the previous timeout when show() is called again", () => {
    const timer = new SuccessTimerState(1000);
    timer.show();
    vi.advanceTimersByTime(500);
    timer.show();
    expect(timer.visible).toBe(true);
    // The original timer would have fired at 1000 ms total; advance past it
    vi.advanceTimersByTime(500);
    // The new timer started at 500 ms; 1000 ms from now is 1500 ms total
    expect(timer.visible).toBe(true);
    vi.advanceTimersByTime(500);
    expect(timer.visible).toBe(false);
  });

  it("clear() sets visible to false immediately", () => {
    const timer = new SuccessTimerState(1000);
    timer.show();
    timer.clear();
    expect(timer.visible).toBe(false);
  });

  it("clear() cancels the pending timeout so visible stays false", () => {
    const timer = new SuccessTimerState(1000);
    timer.show();
    timer.clear();
    vi.advanceTimersByTime(2000);
    expect(timer.visible).toBe(false);
  });

  it("clear() is safe to call when no timeout is pending", () => {
    const timer = new SuccessTimerState();
    expect(() => timer.clear()).not.toThrow();
    expect(timer.visible).toBe(false);
  });
});

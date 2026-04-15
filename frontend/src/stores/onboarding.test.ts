import { describe, it, expect, beforeEach } from "vitest";
import { onboardingStore } from "./onboarding.svelte";

// localStorage is stubbed globally by test-setup.ts before each test.

describe("onboardingStore", () => {
  const userId = "user-abc-123";

  beforeEach(() => {
    // Ensure a clean slate before each test.
    onboardingStore.clearSkip(userId);
  });

  it("isSkipped returns false when no entry exists in localStorage", () => {
    expect(onboardingStore.isSkipped(userId)).toBe(false);
  });

  it("isSkipped returns false when userId is undefined", () => {
    expect(onboardingStore.isSkipped(undefined)).toBe(false);
  });

  it("skip writes the flag for the given userId", () => {
    onboardingStore.skip(userId);
    expect(
      localStorage.getItem(`biblioteka_onboarding_skipped_${userId}`),
    ).toBe("1");
  });

  it("isSkipped returns true after skip is called", () => {
    onboardingStore.skip(userId);
    expect(onboardingStore.isSkipped(userId)).toBe(true);
  });

  it("skip is a no-op when userId is undefined", () => {
    onboardingStore.skip(undefined);
    // Nothing should be written.
    expect(localStorage.length).toBe(0);
  });

  it("clearSkip removes the flag", () => {
    onboardingStore.skip(userId);
    onboardingStore.clearSkip(userId);
    expect(
      localStorage.getItem(`biblioteka_onboarding_skipped_${userId}`),
    ).toBeNull();
  });

  it("isSkipped returns false after clearSkip", () => {
    onboardingStore.skip(userId);
    onboardingStore.clearSkip(userId);
    expect(onboardingStore.isSkipped(userId)).toBe(false);
  });

  it("clearSkip is a no-op when userId is undefined", () => {
    // Should not throw.
    expect(() => onboardingStore.clearSkip(undefined)).not.toThrow();
  });

  it("skip states are isolated per userId", () => {
    const otherUserId = "user-xyz-456";
    onboardingStore.skip(userId);

    expect(onboardingStore.isSkipped(userId)).toBe(true);
    expect(onboardingStore.isSkipped(otherUserId)).toBe(false);
  });
});

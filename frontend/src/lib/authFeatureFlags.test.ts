import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchAuthFeatureFlags } from "./authFeatureFlags";

vi.mock("./api", () => ({
  getOidcEnabled: vi.fn(),
  getSignupEnabled: vi.fn(),
  getPasskeyEnabled: vi.fn(),
}));

import { getOidcEnabled, getSignupEnabled, getPasskeyEnabled } from "./api";

describe("fetchAuthFeatureFlags", () => {
  const signal = new AbortController().signal;

  beforeEach(() => {
    vi.mocked(getOidcEnabled).mockResolvedValue(false);
    vi.mocked(getSignupEnabled).mockResolvedValue(true);
    vi.mocked(getPasskeyEnabled).mockResolvedValue(false);
  });

  it("returns resolved values when all API calls succeed", async () => {
    vi.mocked(getOidcEnabled).mockResolvedValue(true);
    vi.mocked(getSignupEnabled).mockResolvedValue(false);
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags).toEqual({
      oidcEnabled: true,
      signupEnabled: false,
      passkeyEnabled: true,
      initError: null,
    });
  });

  it("sets initError to null when all calls succeed", async () => {
    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags.initError).toBeNull();
  });

  it("falls back to false for oidcEnabled when oidc call fails", async () => {
    vi.mocked(getOidcEnabled).mockRejectedValue(new Error("network error"));

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags.oidcEnabled).toBe(false);
  });

  it("falls back to true for signupEnabled when signup call fails", async () => {
    vi.mocked(getSignupEnabled).mockRejectedValue(new Error("network error"));

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags.signupEnabled).toBe(true);
  });

  it("falls back to false for passkeyEnabled when passkey call fails", async () => {
    vi.mocked(getPasskeyEnabled).mockRejectedValue(new Error("network error"));

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags.passkeyEnabled).toBe(false);
  });

  it("sets initError when any call fails", async () => {
    vi.mocked(getOidcEnabled).mockRejectedValue(new Error("network error"));

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags.initError).toBe(
      "Unable to reach the server to load auth settings",
    );
  });

  it("applies all fallback defaults and sets initError when all calls fail", async () => {
    vi.mocked(getOidcEnabled).mockRejectedValue(new Error("fail"));
    vi.mocked(getSignupEnabled).mockRejectedValue(new Error("fail"));
    vi.mocked(getPasskeyEnabled).mockRejectedValue(new Error("fail"));

    const flags = await fetchAuthFeatureFlags(signal);

    expect(flags).toEqual({
      oidcEnabled: false,
      signupEnabled: true,
      passkeyEnabled: false,
      initError: "Unable to reach the server to load auth settings",
    });
  });
});

import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { copyToClipboard } from "./clipboard";

let originalExecCommand: PropertyDescriptor | undefined;

beforeEach(() => {
  originalExecCommand = Object.getOwnPropertyDescriptor(
    document,
    "execCommand",
  );
});

afterEach(() => {
  if (originalExecCommand) {
    Object.defineProperty(document, "execCommand", originalExecCommand);
  } else {
    delete (document as unknown as Record<string, unknown>)["execCommand"];
  }
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("copyToClipboard", () => {
  it("uses the Clipboard API when available", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
    });

    await copyToClipboard("hello");

    expect(writeText).toHaveBeenCalledWith("hello");
  });

  it("falls back to execCommand when Clipboard API is unavailable", async () => {
    vi.stubGlobal("navigator", {});

    Object.defineProperty(document, "execCommand", {
      value: vi.fn().mockReturnValue(true),
      writable: true,
      configurable: true,
    });

    await copyToClipboard("fallback text");

    expect(document.execCommand).toHaveBeenCalledWith("copy");
  });

  it("throws when execCommand returns false", async () => {
    vi.stubGlobal("navigator", {});

    Object.defineProperty(document, "execCommand", {
      value: vi.fn().mockReturnValue(false),
      writable: true,
      configurable: true,
    });

    await expect(copyToClipboard("fail")).rejects.toThrow(
      "clipboard copy command was rejected",
    );
  });

  it("propagates errors from the Clipboard API", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("permission denied"));
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
    });

    await expect(copyToClipboard("text")).rejects.toThrow("permission denied");
  });
});

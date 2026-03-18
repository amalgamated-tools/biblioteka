import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import * as api from "../lib/api";

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "smtp",
  },
}));

vi.mock("../lib/api", () => ({
  getConfigStatus: vi.fn(),
  getOidcConfig: vi.fn(),
  getSmtpConfig: vi.fn(),
  setSmtpConfig: vi.fn(),
  testSmtpConfig: vi.fn(),
}));

import Settings from "./Settings.svelte";

const baseConfigStatus = {
  oidc_configured: false,
  smtp_configured: true,
  is_admin: true,
};

const baseSmtpConfig = {
  host: "smtp.example.com",
  port: "587",
  username: "",
  from: "noreply@example.com",
  tls: "starttls",
  env_override: false,
  password_set: false,
};

describe("Settings SMTP feedback accessibility", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.mocked(api.getConfigStatus).mockResolvedValue(baseConfigStatus);
    vi.mocked(api.getSmtpConfig).mockResolvedValue(baseSmtpConfig);
  });

  it("announces SMTP success feedback with polite status live regions", async () => {
    vi.mocked(api.testSmtpConfig).mockResolvedValue({ message: "Test email sent" });
    vi.mocked(api.setSmtpConfig).mockResolvedValue({ message: "SMTP settings saved" });

    render(Settings);

    await fireEvent.click(await screen.findByRole("button", { name: /send test email/i }));

    const testSuccess = await screen.findByText("Test email sent");
    expect(testSuccess.closest('[role="status"]')).toHaveAttribute("aria-live", "polite");

    await fireEvent.click(screen.getByRole("button", { name: /update configuration/i }));

    const saveSuccess = await screen.findByText("SMTP settings saved");
    expect(saveSuccess.closest('[role="status"]')).toHaveAttribute("aria-live", "polite");
  });

  it("announces SMTP error feedback with alert live regions", async () => {
    vi.mocked(api.testSmtpConfig).mockRejectedValue(new Error("Test email failed"));
    vi.mocked(api.setSmtpConfig).mockRejectedValue(new Error("SMTP save failed"));

    render(Settings);

    await fireEvent.click(await screen.findByRole("button", { name: /send test email/i }));

    const testError = await screen.findByText("Test email failed");
    expect(testError.closest('[role="alert"]')).toBeInTheDocument();

    await fireEvent.click(screen.getByRole("button", { name: /update configuration/i }));

    const saveError = await screen.findByText("SMTP save failed");
    expect(saveError.closest('[role="alert"]')).toBeInTheDocument();
  });
});

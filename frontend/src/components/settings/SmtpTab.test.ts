import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  getSmtpConfig: vi.fn().mockResolvedValue({
    host: "smtp.example.com",
    port: "587",
    username: "user@example.com",
    password_set: true,
    from: "noreply@example.com",
    tls: "starttls",
    env_override: false,
  }),
  getConfigStatus: vi.fn().mockResolvedValue({
    smtp_configured: true,
    oidc_configured: false,
    is_admin: true,
  }),
  setSmtpConfig: vi
    .fn()
    .mockResolvedValue({ message: "SMTP configuration saved" }),
  testSmtpConfig: vi.fn().mockResolvedValue({ message: "Test email sent" }),
}));

vi.mock("lucide-svelte", () => ({
  Mail: () => {},
  Send: () => {},
}));

import SmtpTab from "./SmtpTab.svelte";
import { getSmtpConfig, setSmtpConfig, testSmtpConfig } from "../../lib/api";

describe("SmtpTab rendering", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the Email / SMTP Configuration heading", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();

    expect(
      screen.getByRole("heading", { name: /Email \/ SMTP Configuration/i }),
    ).toBeInTheDocument();
  });

  it("loads SMTP config on mount", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    expect(getSmtpConfig).toHaveBeenCalled();
  });

  it("shows 'Not configured' status when initialSmtpConfigured is false", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();

    expect(screen.getByText("Not configured")).toBeInTheDocument();
  });

  it("shows 'Configured' status when initialSmtpConfigured is true", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();

    expect(screen.getByText("Configured")).toBeInTheDocument();
  });

  it("shows 'Save Configuration' button when not yet configured", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Save Configuration" }),
    ).toBeInTheDocument();
  });

  it("shows 'Send Test Email' button when already configured", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Send Test Email" }),
    ).toBeInTheDocument();
  });

  it("does not show 'Send Test Email' when not configured", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    expect(
      screen.queryByRole("button", { name: "Send Test Email" }),
    ).toBeNull();
  });

  it("shows env override notice when smtp.env_override is true", async () => {
    vi.mocked(getSmtpConfig).mockResolvedValueOnce({
      host: "smtp.example.com",
      port: "587",
      username: "user@example.com",
      password_set: false,
      from: "noreply@example.com",
      tls: "starttls",
      env_override: true,
    });
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    expect(
      screen.getByText(/configured via environment variables/i),
    ).toBeInTheDocument();
  });
});

describe("SmtpTab form validation", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows error when SMTP Host is empty on submit", async () => {
    vi.mocked(getSmtpConfig).mockResolvedValueOnce({
      host: "",
      port: "587",
      username: "",
      password_set: false,
      from: "",
      tls: "starttls",
      env_override: false,
    });
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "SMTP Host is required",
    );
  });

  it("shows error when From Address is empty on submit", async () => {
    vi.mocked(getSmtpConfig).mockResolvedValueOnce({
      host: "smtp.example.com",
      port: "587",
      username: "",
      password_set: false,
      from: "",
      tls: "starttls",
      env_override: false,
    });
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "From Address is required",
    );
  });

  it("calls setSmtpConfig with correct values on valid submission", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(setSmtpConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        host: "smtp.example.com",
        from: "noreply@example.com",
      }),
    );
  });

  it("shows success message after saving", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByText("SMTP configuration saved")).toBeInTheDocument();
  });

  it("shows error banner when setSmtpConfig rejects", async () => {
    vi.mocked(setSmtpConfig).mockRejectedValueOnce(
      new Error("Connection refused"),
    );
    render(SmtpTab, { props: { initialSmtpConfigured: false } });
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Connection refused");
  });
});

describe("SmtpTab test email", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
    vi.clearAllMocks();
  });

  it("calls testSmtpConfig when Send Test Email is clicked", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: "Send Test Email" }),
    );
    await tick();
    await tick();

    expect(testSmtpConfig).toHaveBeenCalled();
  });

  it("shows test success message when testSmtpConfig resolves", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: "Send Test Email" }),
    );
    await tick();
    await tick();

    expect(screen.getByText("Test email sent")).toBeInTheDocument();
  });

  it("shows test error message when testSmtpConfig rejects", async () => {
    vi.mocked(testSmtpConfig).mockRejectedValueOnce(
      new Error("SMTP unreachable"),
    );
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: "Send Test Email" }),
    );
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("SMTP unreachable");
  });

  it("auto-dismisses the test success message after 5 seconds", async () => {
    render(SmtpTab, { props: { initialSmtpConfigured: true } });
    await tick();
    await tick();

    await fireEvent.click(
      screen.getByRole("button", { name: "Send Test Email" }),
    );
    await tick();
    await tick();

    expect(screen.getByText("Test email sent")).toBeInTheDocument();

    await vi.advanceTimersByTimeAsync(5000);
    await tick();

    expect(screen.queryByText("Test email sent")).toBeNull();
  });
});

import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";

const {
  getConfigStatusMock,
  getOidcConfigMock,
  getSmtpConfigMock,
  setSmtpConfigMock,
  testSmtpConfigMock,
} = vi.hoisted(() => ({
  getConfigStatusMock: vi.fn().mockResolvedValue({
    oidc_configured: false,
    smtp_configured: true,
    is_admin: true,
  }),
  getOidcConfigMock: vi.fn().mockResolvedValue(null),
  getSmtpConfigMock: vi.fn().mockResolvedValue({
    host: "",
    port: "587",
    username: "",
    from: "",
    tls: "starttls",
    env_override: false,
    password_set: false,
  }),
  setSmtpConfigMock: vi.fn().mockResolvedValue({ message: "Saved" }),
  testSmtpConfigMock: vi.fn(),
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "smtp",
  },
}));

vi.mock("../lib/api", () => ({
  getConfigStatus: getConfigStatusMock,
  getOidcConfig: getOidcConfigMock,
  getSmtpConfig: getSmtpConfigMock,
  setSmtpConfig: setSmtpConfigMock,
  testSmtpConfig: testSmtpConfigMock,
}));

vi.mock("lucide-svelte", () => ({
  Mail: () => {},
  Palette: () => {},
  Shield: () => {},
  Users: () => {},
  Send: () => {},
  KeyRound: () => {},
  BookOpen: () => {},
}));

vi.mock("./settings/AccountTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/OidcTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/UsersTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/PreferencesTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/APIKeysTab.svelte", () => ({ default: () => {} }));
vi.mock("./settings/KoboTab.svelte", () => ({ default: () => {} }));

import Settings from "./Settings.svelte";

describe("Settings accessibility", () => {
  it("announces SMTP validation errors and save success messages", async () => {
    const { container } = render(Settings);

    await waitFor(() => {
      expect(screen.getByText("Email / SMTP Configuration")).toBeInTheDocument();
    });

    expect(
      container.querySelectorAll('[aria-live="polite"][aria-atomic="true"]'),
    ).toHaveLength(2); // one for smtpTestMessage, one for smtpSuccessMessage
    expect(
      container.querySelectorAll('[aria-live="assertive"][aria-atomic="true"]'),
    ).toHaveLength(2); // one for smtpTestError, one for smtpError

    const form = container.querySelector("form");
    expect(form).toBeInTheDocument();
    await fireEvent.submit(form!);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "SMTP Host is required",
      );
    });

    const host = container.querySelector("#smtp-host") as HTMLInputElement;
    const from = container.querySelector("#smtp-from") as HTMLInputElement;
    await fireEvent.input(host, { target: { value: "smtp.example.com" } });
    await fireEvent.input(from, { target: { value: "noreply@example.com" } });
    await fireEvent.submit(form!);

    await waitFor(() => {
      expect(screen.getByRole("status")).toHaveTextContent("Saved");
    });
  });
});

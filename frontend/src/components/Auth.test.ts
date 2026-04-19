import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";
import { authStore } from "../stores/auth.svelte";

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    loading: false,
    user: null,
    signIn: vi.fn(),
    signUp: vi.fn(),
  },
}));

vi.mock("../lib/api", () => ({
  getOidcEnabled: vi.fn().mockResolvedValue(false),
  getSignupEnabled: vi.fn().mockResolvedValue(true),
  getPasskeyEnabled: vi.fn().mockResolvedValue(false),
  beginPasskeyLogin: vi.fn(),
  finishPasskeyLogin: vi.fn(),
}));

vi.mock("lucide-svelte", () => ({ BookCheck: () => {} }));

import { getOidcEnabled, getSignupEnabled } from "../lib/api";

import Auth from "./Auth.svelte";

describe("Auth", () => {
  afterEach(() => {
    cleanup();
  });

  const renderAuth = async () => {
    render(Auth);
    await tick();
  };

  it("has a main landmark region", async () => {
    await renderAuth();

    const main = screen.getByRole("main");
    expect(main).toBeInTheDocument();
  });

  it("renders tab buttons with correct ARIA attributes", async () => {
    await renderAuth();

    const tabList = screen.getByRole("tablist");
    expect(tabList).toBeInTheDocument();

    const loginTab = screen.getByRole("tab", { name: "Login" });
    expect(loginTab).toHaveAttribute("aria-selected", "true");
    expect(loginTab).toHaveAttribute("aria-controls", "login-panel");

    const signupTab = screen.getByRole("tab", { name: "Sign Up" });
    expect(signupTab).toHaveAttribute("aria-selected", "false");
    expect(signupTab).toHaveAttribute("aria-controls", "signup-panel");
  });

  it("renders tab panels with correct ARIA attributes", async () => {
    await renderAuth();

    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    expect(panels).toHaveLength(2);

    const loginPanel = panels.find((p) => p.id === "login-panel");
    expect(loginPanel).toBeDefined();
    expect(loginPanel!).toHaveAttribute("aria-labelledby", "login-tab");

    const signupPanel = panels.find((p) => p.id === "signup-panel");
    expect(signupPanel).toBeDefined();
    expect(signupPanel!).toHaveAttribute("aria-labelledby", "signup-tab");
  });

  it("switches ARIA state when the Sign Up tab is clicked", async () => {
    await renderAuth();

    const user = userEvent.setup();
    const signupTab = screen.getByRole("tab", { name: "Sign Up" });
    await user.click(signupTab);
    await tick();

    expect(signupTab).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Login" })).toHaveAttribute(
      "aria-selected",
      "false",
    );

    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    const signupPanel = panels.find((p) => p.id === "signup-panel");
    expect(signupPanel).toBeDefined();
    expect(signupPanel!).not.toHaveAttribute("hidden");
  });

  it("hides the Sign Up tab and panel when signup is disabled", async () => {
    vi.mocked(getSignupEnabled).mockResolvedValueOnce(false);

    render(Auth);
    await vi.waitFor(() => {
      expect(screen.queryByRole("tab", { name: "Sign Up" })).toBeNull();
    });

    const loginTab = screen.getByRole("tab", { name: "Login" });
    expect(loginTab).toBeInTheDocument();

    // Signup panel should not be rendered when signup is disabled
    const panels = screen.getAllByRole("tabpanel", { hidden: true });
    expect(panels).toHaveLength(1);
    expect(panels[0].id).toBe("login-panel");
  });

  it("keyboard navigation does not hide login panel when signup is disabled", async () => {
    vi.mocked(getSignupEnabled).mockResolvedValueOnce(false);

    const user = userEvent.setup();
    render(Auth);
    await vi.waitFor(() => {
      expect(screen.queryByRole("tab", { name: "Sign Up" })).toBeNull();
    });

    const loginTab = screen.getByRole("tab", { name: "Login" });
    loginTab.focus();

    // ArrowRight should not toggle away from login
    await user.keyboard("{ArrowRight}");
    await tick();

    const loginPanel = screen.getByRole("tabpanel", { hidden: true });
    expect(loginPanel.id).toBe("login-panel");
    expect(loginPanel).not.toHaveAttribute("hidden");
  });

  it("shows an error banner when OIDC check fails", async () => {
    vi.mocked(getOidcEnabled).mockRejectedValueOnce(new Error("network"));

    render(Auth);
    await vi.waitFor(() => {
      expect(
        screen.getByText("Unable to reach the server to load auth settings"),
      ).toBeInTheDocument();
    });
  });

  it("shows an error banner when signup check fails", async () => {
    vi.mocked(getSignupEnabled).mockRejectedValueOnce(new Error("network"));

    render(Auth);
    await vi.waitFor(() => {
      expect(
        screen.getByText("Unable to reach the server to load auth settings"),
      ).toBeInTheDocument();
    });
  });

  it("marks login fields invalid and associates them with the error banner", async () => {
    vi.mocked(authStore.signIn).mockResolvedValueOnce({
      error: new Error("Invalid credentials"),
    });

    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);
    await user.type(loginEmail, "you@example.com");
    await user.type(loginPassword, "securepass");
    await user.click(
      within(loginPanel).getByRole("button", { name: "Sign In" }),
    );
    await tick();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveAttribute("id", "login-auth-error");

    expect(loginEmail).toHaveAttribute("aria-invalid", "true");
    expect(loginEmail).toHaveAttribute("aria-describedby", "login-auth-error");
    expect(loginPassword).toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
  });

  it("marks login and signup fields as required and shows required-field notes", async () => {
    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginLegend = within(loginPanel).getByText(/are required/i, {
      selector: "p",
    });
    const loginAsterisk = loginLegend.querySelector('span[aria-hidden="true"]');
    expect(loginAsterisk).toBeInTheDocument();
    expect(loginAsterisk).toHaveTextContent("*");
    const loginSrOnly = loginLegend.querySelector(".sr-only");
    expect(loginSrOnly).toBeInTheDocument();
    expect(loginSrOnly).toHaveTextContent("an asterisk");

    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);
    expect(loginEmail).toHaveAttribute("required");
    expect(loginEmail).toHaveAttribute("aria-required", "true");
    expect(loginPassword).toHaveAttribute("required");
    expect(loginPassword).toHaveAttribute("aria-required", "true");

    await user.click(screen.getByRole("tab", { name: "Sign Up" }));
    await tick();

    const signupPanel = screen.getByRole("tabpanel", { name: "Sign Up" });
    const signupLegend = within(signupPanel).getByText(/are required/i, {
      selector: "p",
    });
    const signupAsterisk = signupLegend.querySelector(
      'span[aria-hidden="true"]',
    );
    expect(signupAsterisk).toBeInTheDocument();
    expect(signupAsterisk).toHaveTextContent("*");
    const signupSrOnly = signupLegend.querySelector(".sr-only");
    expect(signupSrOnly).toBeInTheDocument();
    expect(signupSrOnly).toHaveTextContent("an asterisk");

    const signupName = within(signupPanel).getByLabelText(/Name/i);
    const signupEmail = within(signupPanel).getByLabelText(/Email/i);
    const signupPassword = within(signupPanel).getByLabelText(/Password/i);
    expect(signupName).toHaveAttribute("required");
    expect(signupName).toHaveAttribute("aria-required", "true");
    expect(signupEmail).toHaveAttribute("required");
    expect(signupEmail).toHaveAttribute("aria-required", "true");
    expect(signupPassword).toHaveAttribute("required");
    expect(signupPassword).toHaveAttribute("aria-required", "true");
  });

  it("marks only the password field invalid for a password-only login error", async () => {
    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);

    await user.type(loginEmail, "reader@example.com");
    await user.type(loginPassword, "12345");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Password must be at least 6 characters");

    expect(loginEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(loginEmail).not.toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
    expect(loginPassword).toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
  });

  it("does not mark login fields invalid for generic server errors", async () => {
    vi.mocked(authStore.signIn).mockResolvedValueOnce({
      error: new Error("Authentication failed"),
    });

    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);

    await user.type(loginEmail, "reader@example.com");
    await user.type(loginPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Authentication failed");

    expect(loginEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(loginEmail).not.toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
    expect(loginPassword).not.toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).not.toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
  });

  it("does not mark login fields invalid for ambiguous credential errors", async () => {
    vi.mocked(authStore.signIn).mockResolvedValueOnce({
      error: new Error("Invalid email or password"),
    });

    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);

    await user.type(loginEmail, "reader@example.com");
    await user.type(loginPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    await screen.findByRole("alert");

    expect(loginEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).not.toHaveAttribute("aria-invalid", "true");
  });

  it("marks only the email field invalid for an email-specific server error", async () => {
    vi.mocked(authStore.signIn).mockResolvedValueOnce({
      error: new Error("Email is not valid"),
    });

    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);

    await user.type(loginEmail, "reader@example.com");
    await user.type(loginPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    await screen.findByRole("alert");

    expect(loginEmail).toHaveAttribute("aria-invalid", "true");
    expect(loginEmail).toHaveAttribute("aria-describedby", "login-auth-error");
    expect(loginPassword).not.toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).not.toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
  });

  it("marks only the password field invalid for a password-specific server error", async () => {
    vi.mocked(authStore.signIn).mockResolvedValueOnce({
      error: new Error("Incorrect password"),
    });

    const user = userEvent.setup();
    await renderAuth();

    const loginPanel = screen.getByRole("tabpanel", { name: "Login" });
    const loginEmail = within(loginPanel).getByLabelText(/Email/i);
    const loginPassword = within(loginPanel).getByLabelText(/Password/i);

    await user.type(loginEmail, "reader@example.com");
    await user.type(loginPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    await screen.findByRole("alert");

    expect(loginEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(loginEmail).not.toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
    expect(loginPassword).toHaveAttribute("aria-invalid", "true");
    expect(loginPassword).toHaveAttribute(
      "aria-describedby",
      "login-auth-error",
    );
  });

  it("marks only the password field invalid for a password-only signup error", async () => {
    const user = userEvent.setup();
    await renderAuth();

    await user.click(screen.getByRole("tab", { name: "Sign Up" }));
    await tick();

    const signupPanel = screen.getByRole("tabpanel", { name: "Sign Up" });
    const signupName = within(signupPanel).getByLabelText(/Name/i);
    const signupEmail = within(signupPanel).getByLabelText(/Email/i);
    const signupPassword = within(signupPanel).getByLabelText(/Password/i);

    await user.type(signupName, "Reader");
    await user.type(signupEmail, "reader@example.com");
    await user.type(signupPassword, "12345");
    await user.click(screen.getByRole("button", { name: "Create Account" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Password must be at least 6 characters");

    expect(signupName).not.toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(signupPassword).toHaveAttribute("aria-invalid", "true");
    expect(signupPassword).toHaveAttribute(
      "aria-describedby",
      "signup-auth-error",
    );
  });

  it("does not mark signup fields invalid for generic signup server errors", async () => {
    vi.mocked(authStore.signUp).mockResolvedValueOnce({
      error: new Error("Signup failed"),
    });

    const user = userEvent.setup();
    await renderAuth();

    await user.click(screen.getByRole("tab", { name: "Sign Up" }));
    await tick();

    const signupPanel = screen.getByRole("tabpanel", { name: "Sign Up" });
    const signupName = within(signupPanel).getByLabelText(/Name/i);
    const signupEmail = within(signupPanel).getByLabelText(/Email/i);
    const signupPassword = within(signupPanel).getByLabelText(/Password/i);

    await user.type(signupName, "Reader");
    await user.type(signupEmail, "reader@example.com");
    await user.type(signupPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Create Account" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Signup failed");

    expect(signupName).not.toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).not.toHaveAttribute("aria-invalid", "true");
    expect(signupPassword).not.toHaveAttribute("aria-invalid", "true");
  });

  it("marks only the email field invalid for a duplicate-email signup error", async () => {
    vi.mocked(authStore.signUp).mockResolvedValueOnce({
      error: new Error("email already exists"),
    });

    const user = userEvent.setup();
    await renderAuth();

    await user.click(screen.getByRole("tab", { name: "Sign Up" }));
    await tick();

    const signupPanel = screen.getByRole("tabpanel", { name: "Sign Up" });
    const signupName = within(signupPanel).getByLabelText(/Name/i);
    const signupEmail = within(signupPanel).getByLabelText(/Email/i);
    const signupPassword = within(signupPanel).getByLabelText(/Password/i);

    await user.type(signupName, "Reader");
    await user.type(signupEmail, "reader@example.com");
    await user.type(signupPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Create Account" }));

    await screen.findByRole("alert");

    expect(signupName).not.toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).toHaveAttribute(
      "aria-describedby",
      "signup-auth-error",
    );
    expect(signupPassword).not.toHaveAttribute("aria-invalid", "true");
  });

  it("marks only the email field invalid for an email-specific signup server error", async () => {
    vi.mocked(authStore.signUp).mockResolvedValueOnce({
      error: new Error("Email is not valid"),
    });

    const user = userEvent.setup();
    await renderAuth();

    await user.click(screen.getByRole("tab", { name: "Sign Up" }));
    await tick();

    const signupPanel = screen.getByRole("tabpanel", { name: "Sign Up" });
    const signupName = within(signupPanel).getByLabelText(/Name/i);
    const signupEmail = within(signupPanel).getByLabelText(/Email/i);
    const signupPassword = within(signupPanel).getByLabelText(/Password/i);

    await user.type(signupName, "Reader");
    await user.type(signupEmail, "reader@example.com");
    await user.type(signupPassword, "securepass");
    await user.click(screen.getByRole("button", { name: "Create Account" }));

    await screen.findByRole("alert");

    expect(signupName).not.toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).toHaveAttribute("aria-invalid", "true");
    expect(signupEmail).toHaveAttribute(
      "aria-describedby",
      "signup-auth-error",
    );
    expect(signupPassword).not.toHaveAttribute("aria-invalid", "true");
  });
});

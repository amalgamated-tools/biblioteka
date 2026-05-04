import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import {
  cleanup,
  render,
  screen,
  waitFor,
  fireEvent,
} from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: { id: "1", email: "admin@example.com" },
  },
}));

vi.mock("../../lib/api", () => ({
  listUsers: vi.fn().mockResolvedValue([]),
  setUserAdmin: vi.fn(),
  getRegistrationConfig: vi
    .fn()
    .mockResolvedValue({ registration_disabled: false }),
  setRegistrationConfig: vi.fn(),
}));

vi.mock("lucide-svelte", () => ({
  Users: () => {},
  UserX: () => {},
}));

import UsersTab from "./UsersTab.svelte";
import {
  listUsers,
  getRegistrationConfig,
  setRegistrationConfig,
} from "../../lib/api";

describe("UsersTab accessibility", () => {
  afterEach(() => {
    cleanup();
  });

  it("labels the users table with aria-label (WCAG 1.3.1)", () => {
    render(UsersTab, {
      props: {
        cachedUsers: [
          {
            id: "1",
            name: "Admin User",
            email: "admin@example.com",
            is_admin: true,
            oidc_linked: false,
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        onUsersLoaded: () => {},
      },
    });

    expect(screen.getByRole("table", { name: "Users" })).toBeInTheDocument();
  });

  it("marks each table header as a column header", () => {
    render(UsersTab, {
      props: {
        cachedUsers: [
          {
            id: "1",
            name: "Admin User",
            email: "admin@example.com",
            is_admin: true,
            oidc_linked: false,
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        onUsersLoaded: () => {},
      },
    });

    for (const name of ["Name", "Email", "Type", "Role", "Joined"]) {
      expect(screen.getByRole("columnheader", { name })).toHaveAttribute(
        "scope",
        "col",
      );
    }
  });

  it("gives toggle-admin buttons descriptive accessible names", () => {
    render(UsersTab, {
      props: {
        cachedUsers: [
          {
            id: "1",
            name: "Admin User",
            email: "admin@example.com",
            is_admin: true,
            oidc_linked: false,
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "2",
            name: "Reader User",
            email: "reader@example.com",
            is_admin: false,
            oidc_linked: false,
            created_at: "2026-01-02T00:00:00Z",
          },
          {
            id: "3",
            name: "Staff User",
            email: "staff@example.com",
            is_admin: true,
            oidc_linked: false,
            created_at: "2026-01-03T00:00:00Z",
          },
        ],
        onUsersLoaded: () => {},
      },
    });

    expect(
      screen.getByRole("button", { name: "Grant admin role to Reader User" }),
    ).toHaveTextContent("User");
    expect(
      screen.getByRole("button", { name: "Remove admin role from Staff User" }),
    ).toHaveTextContent("Admin");
  });

  it("calls listUsers exactly once when cachedUsers is empty", async () => {
    vi.mocked(listUsers).mockClear();

    render(UsersTab, {
      props: {
        cachedUsers: [],
        onUsersLoaded: vi.fn(),
      },
    });

    await waitFor(() => {
      expect(vi.mocked(listUsers)).toHaveBeenCalledOnce();
    });
  });
});

describe("UsersTab registration config", () => {
  beforeEach(() => {
    vi.mocked(getRegistrationConfig).mockResolvedValue({
      registration_disabled: false,
    });
    vi.mocked(setRegistrationConfig).mockResolvedValue({
      registration_disabled: false,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  const defaultProps = {
    cachedUsers: [],
    onUsersLoaded: vi.fn(),
  };

  it("shows 'Registration is enabled' when config returns registration_disabled: false", async () => {
    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    expect(screen.getByText("Registration is enabled")).toBeInTheDocument();
  });

  it("shows 'Registration is disabled' when config returns registration_disabled: true", async () => {
    vi.mocked(getRegistrationConfig).mockResolvedValue({
      registration_disabled: true,
    });

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    expect(screen.getByText("Registration is disabled")).toBeInTheDocument();
  });

  it("shows 'Disable Registration' button when registration is enabled", async () => {
    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Disable Registration" }),
    ).toBeInTheDocument();
  });

  it("shows 'Enable Registration' button when registration is disabled", async () => {
    vi.mocked(getRegistrationConfig).mockResolvedValue({
      registration_disabled: true,
    });

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Enable Registration" }),
    ).toBeInTheDocument();
  });

  it("calls setRegistrationConfig with registration_disabled: true when disabling", async () => {
    vi.mocked(setRegistrationConfig).mockResolvedValue({
      registration_disabled: true,
    });

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    fireEvent.click(
      screen.getByRole("button", { name: "Disable Registration" }),
    );
    await tick();
    await tick();

    expect(vi.mocked(setRegistrationConfig)).toHaveBeenCalledWith({
      registration_disabled: true,
    });
  });

  it("shows success message after disabling registration", async () => {
    vi.mocked(setRegistrationConfig).mockResolvedValue({
      registration_disabled: true,
    });

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    fireEvent.click(
      screen.getByRole("button", { name: "Disable Registration" }),
    );
    await tick();
    await tick();

    expect(
      screen.getByText("Public registration disabled."),
    ).toBeInTheDocument();
  });

  it("calls setRegistrationConfig with registration_disabled: false when enabling", async () => {
    vi.mocked(getRegistrationConfig).mockResolvedValue({
      registration_disabled: true,
    });
    vi.mocked(setRegistrationConfig).mockResolvedValue({
      registration_disabled: false,
    });

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    fireEvent.click(
      screen.getByRole("button", { name: "Enable Registration" }),
    );
    await tick();
    await tick();

    expect(vi.mocked(setRegistrationConfig)).toHaveBeenCalledWith({
      registration_disabled: false,
    });
  });

  it("shows error banner when setRegistrationConfig rejects", async () => {
    vi.mocked(setRegistrationConfig).mockRejectedValue(
      new Error("Server error"),
    );

    render(UsersTab, { props: defaultProps });
    await tick();
    await tick();

    fireEvent.click(
      screen.getByRole("button", { name: "Disable Registration" }),
    );
    await tick();
    await tick();

    expect(screen.getByText("Server error")).toBeInTheDocument();
  });
});

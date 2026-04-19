import {
  describe,
  expect,
  it,
  vi,
  afterEach,
  beforeEach,
  beforeAll,
} from "vitest";
import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/svelte";
import { tick } from "svelte";

// PublicKeyCredential is not available in JSDOM; define a minimal stub so that
// `credential instanceof PublicKeyCredential` checks inside PasskeysSection work.
class MockPublicKeyCredential {
  type = "public-key";
  toJSON() {
    return { id: "mock-cred-id" };
  }
}

vi.mock("../../lib/api", () => ({
  getPasskeyEnabled: vi.fn().mockResolvedValue(false),
  listPasskeyCredentials: vi.fn().mockResolvedValue([]),
  deletePasskeyCredential: vi.fn().mockResolvedValue(undefined),
  prepareCreationOptions: vi.fn().mockReturnValue({}),
  beginPasskeyRegistration: vi.fn().mockResolvedValue({
    session_id: "sess-1",
    options: {},
  }),
  finishPasskeyRegistration: vi.fn().mockResolvedValue({
    id: "pk-1",
    name: "My Key",
    aaguid: "aaguid-1",
    created_at: "2026-01-01T00:00:00Z",
  }),
}));

vi.mock("lucide-svelte", () => ({
  KeyRound: () => {},
  Trash2: () => {},
}));

import PasskeysSection from "./PasskeysSection.svelte";
import {
  getPasskeyEnabled,
  listPasskeyCredentials,
  deletePasskeyCredential,
  beginPasskeyRegistration,
  finishPasskeyRegistration,
} from "../../lib/api";

describe("PasskeysSection", () => {
  beforeAll(() => {
    // Stub PublicKeyCredential globally so instanceof checks work.
    Object.defineProperty(globalThis, "PublicKeyCredential", {
      value: MockPublicKeyCredential,
      writable: true,
      configurable: true,
    });

    // Stub navigator.credentials so vi.spyOn can target it.
    Object.defineProperty(navigator, "credentials", {
      value: { create: vi.fn() },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("hides the passkeys section when passkeys are disabled", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(false);
    render(PasskeysSection);

    await waitFor(() => {
      expect(getPasskeyEnabled).toHaveBeenCalledTimes(1);
    });

    expect(
      screen.queryByRole("heading", { name: "Passkeys" }),
    ).not.toBeInTheDocument();
  });

  it("renders the Passkeys section when passkeys are enabled", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    expect(
      screen.getByRole("heading", { name: "Passkeys" }),
    ).toBeInTheDocument();
  });

  it("loads passkey credentials when passkeys are enabled", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    expect(listPasskeyCredentials).toHaveBeenCalledTimes(1);
  });

  it("does not load credentials when passkeys are disabled", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(false);

    render(PasskeysSection);
    await waitFor(() => {
      expect(getPasskeyEnabled).toHaveBeenCalledTimes(1);
    });

    expect(listPasskeyCredentials).not.toHaveBeenCalled();
  });

  it("renders the passkey name input and Add Passkey button when enabled", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    expect(screen.getByLabelText("Passkey name")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add Passkey" }),
    ).toBeInTheDocument();
  });

  it("shows validation error when passkey name is empty on submit", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Passkey name is required",
    );
  });

  it("calls beginPasskeyRegistration with the passkey name on submit", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    const mockCredential = new MockPublicKeyCredential();
    vi.spyOn(navigator.credentials, "create").mockResolvedValue(
      mockCredential as unknown as Credential,
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My iPhone" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(beginPasskeyRegistration).toHaveBeenCalledWith("My iPhone");
  });

  it("shows success message after passkey registration", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    const mockCredential = new MockPublicKeyCredential();
    vi.spyOn(navigator.credentials, "create").mockResolvedValue(
      mockCredential as unknown as Credential,
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My iPhone" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("Passkey registered successfully"),
    ).toBeInTheDocument();
  });

  it("clears passkey name input after successful registration", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    const mockCredential = new MockPublicKeyCredential();
    vi.spyOn(navigator.credentials, "create").mockResolvedValue(
      mockCredential as unknown as Credential,
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My iPhone" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByLabelText("Passkey name")).toHaveValue("");
  });

  it("shows error when navigator.credentials.create returns null", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    vi.spyOn(navigator.credentials, "create").mockResolvedValue(null);

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My Device" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "No passkey was created",
    );
  });

  it("shows error when beginPasskeyRegistration rejects", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);
    vi.mocked(beginPasskeyRegistration).mockRejectedValueOnce(
      new Error("Registration failed"),
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My Device" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Registration failed");
  });

  it("does not show an error when user cancels the passkey dialog (NotAllowedError)", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    const notAllowedError = new DOMException(
      "User cancelled",
      "NotAllowedError",
    );
    vi.spyOn(navigator.credentials, "create").mockRejectedValue(
      notAllowedError,
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My Device" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("calls finishPasskeyRegistration after creating a credential", async () => {
    vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
    vi.mocked(listPasskeyCredentials).mockResolvedValue([]);

    const credJSON = { id: "cred-id" };
    const mockCredential = new MockPublicKeyCredential();
    mockCredential.toJSON = () => credJSON;
    vi.spyOn(navigator.credentials, "create").mockResolvedValue(
      mockCredential as unknown as Credential,
    );

    render(PasskeysSection);
    await screen.findByRole("heading", { name: "Passkeys" });

    await fireEvent.input(screen.getByLabelText("Passkey name"), {
      target: { value: "My iPhone" },
    });

    const form = screen.getByRole("form", { name: "Register new passkey" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(finishPasskeyRegistration).toHaveBeenCalledWith("sess-1", credJSON);
  });

  describe("passkey deletion", () => {
    beforeEach(() => {
      vi.mocked(getPasskeyEnabled).mockResolvedValue(true);
      vi.mocked(listPasskeyCredentials).mockResolvedValue([
        {
          id: "pk-1",
          name: "Laptop",
          aaguid: "aaguid-1",
          created_at: "2026-01-01T00:00:00Z",
        },
      ]);
    });

    it("calls deletePasskeyCredential when delete button is clicked", async () => {
      render(PasskeysSection);

      const deleteButton = await screen.findByRole("button", {
        name: "Delete passkey Laptop",
      });
      await fireEvent.click(deleteButton);
      await tick();
      await tick();

      expect(deletePasskeyCredential).toHaveBeenCalledWith("pk-1");
    });

    it("shows error banner when deletePasskeyCredential rejects", async () => {
      vi.mocked(deletePasskeyCredential).mockRejectedValueOnce(
        new Error("Delete failed"),
      );

      render(PasskeysSection);

      const deleteButton = await screen.findByRole("button", {
        name: "Delete passkey Laptop",
      });
      await fireEvent.click(deleteButton);
      await tick();
      await tick();

      expect(screen.getByRole("alert")).toHaveTextContent("Delete failed");
    });
  });
});

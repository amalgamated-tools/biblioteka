import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  setOidcConfig: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("lucide-svelte", () => ({ Shield: () => {} }));

import OidcTab from "./OidcTab.svelte";
import { setOidcConfig } from "../../lib/api";

const defaultProps = {
  initialOidcConfigured: false,
  initialIssuerUrl: "",
  initialClientId: "",
  initialRedirectUri: "",
  onOidcSaved: vi.fn(),
};

describe("OidcTab", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the OIDC / Single Sign-On heading", () => {
    render(OidcTab, { props: defaultProps });
    expect(
      screen.getByRole("heading", { name: /OIDC \/ Single Sign-On/i }),
    ).toBeInTheDocument();
  });

  it("shows 'Not configured' status badge when not configured", () => {
    render(OidcTab, { props: defaultProps });
    expect(screen.getByText("Not configured")).toBeInTheDocument();
  });

  it("shows 'Configured' status badge when already configured", () => {
    render(OidcTab, {
      props: { ...defaultProps, initialOidcConfigured: true },
    });
    expect(screen.getByText("Configured")).toBeInTheDocument();
  });

  it("shows 'Save Configuration' button label when not yet configured", () => {
    render(OidcTab, { props: defaultProps });
    expect(
      screen.getByRole("button", { name: "Save Configuration" }),
    ).toBeInTheDocument();
  });

  it("shows 'Update Configuration' button label when already configured", () => {
    render(OidcTab, {
      props: { ...defaultProps, initialOidcConfigured: true },
    });
    expect(
      screen.getByRole("button", { name: "Update Configuration" }),
    ).toBeInTheDocument();
  });

  it("shows validation error when Issuer URL is empty on submit", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Issuer URL is required");
  });

  it("shows validation error when Client ID is empty on submit", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    const issuerInput = screen.getByLabelText("Issuer URL");
    await fireEvent.input(issuerInput, {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Client ID is required");
  });

  it("shows validation error when Client Secret is empty on initial save", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Client Secret is required");
  });

  it("shows validation error when Redirect URI is empty on submit", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.input(screen.getByLabelText("Client Secret"), {
      target: { value: "my-secret" },
    });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Redirect URI is required");
  });

  it("does not require Client Secret when already configured", async () => {
    render(OidcTab, {
      props: {
        ...defaultProps,
        initialOidcConfigured: true,
        initialIssuerUrl: "https://auth.example.com",
        initialClientId: "my-client",
        initialRedirectUri: "https://app.example.com/callback",
        onOidcSaved: vi.fn(),
      },
    });

    const form = document.querySelector("form")!;
    // No client secret entered — should not produce a "Client Secret is required" error
    await fireEvent.submit(form);
    await tick();

    expect(setOidcConfig).toHaveBeenCalled();
  });

  it("calls setOidcConfig with correct values on valid submission", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.input(screen.getByLabelText("Client Secret"), {
      target: { value: "my-secret" },
    });
    await fireEvent.input(screen.getByLabelText("Redirect URI"), {
      target: { value: "https://app.example.com/callback" },
    });
    await fireEvent.submit(form);
    await tick();

    expect(setOidcConfig).toHaveBeenCalledWith({
      issuer_url: "https://auth.example.com",
      client_id: "my-client",
      client_secret: "my-secret",
      redirect_uri: "https://app.example.com/callback",
    });
  });

  it("shows success banner after saving", async () => {
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.input(screen.getByLabelText("Client Secret"), {
      target: { value: "my-secret" },
    });
    await fireEvent.input(screen.getByLabelText("Redirect URI"), {
      target: { value: "https://app.example.com/callback" },
    });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("OIDC configuration saved successfully"),
    ).toBeInTheDocument();
  });

  it("calls onOidcSaved callback after successful save", async () => {
    const onOidcSaved = vi.fn();
    render(OidcTab, { props: { ...defaultProps, onOidcSaved } });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.input(screen.getByLabelText("Client Secret"), {
      target: { value: "secret" },
    });
    await fireEvent.input(screen.getByLabelText("Redirect URI"), {
      target: { value: "https://app.example.com/callback" },
    });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(onOidcSaved).toHaveBeenCalledWith(
      expect.objectContaining({ configured: true }),
    );
  });

  it("shows an error banner when setOidcConfig rejects", async () => {
    vi.mocked(setOidcConfig).mockRejectedValueOnce(new Error("Server error"));
    render(OidcTab, { props: defaultProps });

    const form = document.querySelector("form")!;
    await fireEvent.input(screen.getByLabelText("Issuer URL"), {
      target: { value: "https://auth.example.com" },
    });
    await fireEvent.input(screen.getByLabelText("Client ID"), {
      target: { value: "my-client" },
    });
    await fireEvent.input(screen.getByLabelText("Client Secret"), {
      target: { value: "secret" },
    });
    await fireEvent.input(screen.getByLabelText("Redirect URI"), {
      target: { value: "https://app.example.com/callback" },
    });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Server error");
  });
});

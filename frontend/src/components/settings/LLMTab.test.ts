import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  getLLMConfig: vi.fn(),
  setLLMConfig: vi.fn(),
}));

vi.mock("lucide-svelte", () => ({ BrainCircuit: () => {} }));

import LLMTab from "./LLMTab.svelte";
import { getLLMConfig, setLLMConfig } from "../../lib/api";

const defaultConfig = {
  provider: "ollama" as const,
  endpoint: "https://ollama.example.com",
  model: "llama3",
  enabled: true,
};

describe("LLMTab", () => {
  beforeEach(() => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      restart_required: false,
    });
    vi.mocked(setLLMConfig).mockResolvedValue({
      ...defaultConfig,
      restart_required: false,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the LLM Configuration heading", async () => {
    render(LLMTab);
    await tick();

    expect(
      screen.getByRole("heading", { name: /LLM Configuration/i }),
    ).toBeInTheDocument();
  });

  it("loads initial config from getLLMConfig on mount", async () => {
    render(LLMTab);
    await tick();
    await tick();

    expect(vi.mocked(getLLMConfig)).toHaveBeenCalledOnce();
  });

  it("shows 'Enabled' status badge when saved config is enabled with endpoint and model", async () => {
    render(LLMTab);
    await tick();
    await tick();

    expect(screen.getByText("Enabled")).toBeInTheDocument();
  });

  it("shows 'Disabled' status badge when saved config is disabled", async () => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      enabled: false,
    });
    render(LLMTab);
    await tick();
    await tick();

    expect(screen.getByText("Disabled")).toBeInTheDocument();
  });

  it("shows 'Disabled' badge when saved config has no model even if enabled", async () => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      model: "",
    });
    render(LLMTab);
    await tick();
    await tick();

    expect(screen.getByText("Disabled")).toBeInTheDocument();
  });

  it("renders endpoint and model inputs", async () => {
    render(LLMTab);
    await tick();

    expect(screen.getByLabelText("Endpoint")).toBeInTheDocument();
    expect(screen.getByLabelText("Model")).toBeInTheDocument();
  });

  it("shows validation error when endpoint is empty and LLM is enabled on submit", async () => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      endpoint: "",
    });
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Endpoint is required when LLM is enabled",
    );
  });

  it("shows validation error when model is empty and LLM is enabled on submit", async () => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      model: "",
    });
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Model is required when LLM is enabled",
    );
  });

  it("skips validation when LLM is disabled", async () => {
    vi.mocked(getLLMConfig).mockResolvedValue({
      ...defaultConfig,
      enabled: false,
      endpoint: "",
      model: "",
    });
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(vi.mocked(setLLMConfig)).toHaveBeenCalled();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("shows success banner after saving when no restart is required", async () => {
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("LLM configuration saved successfully."),
    ).toBeInTheDocument();
  });

  it("hides success banner and shows restart banner when restart is required", async () => {
    vi.mocked(setLLMConfig).mockResolvedValue({
      ...defaultConfig,
      restart_required: true,
    });
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.queryByText("LLM configuration saved successfully."),
    ).toBeNull();
    expect(screen.getByText(/server restart is required/i)).toBeInTheDocument();
  });

  it("shows error banner when setLLMConfig rejects", async () => {
    vi.mocked(setLLMConfig).mockRejectedValueOnce(new Error("Bad endpoint"));
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Bad endpoint");
  });

  it("calls setLLMConfig with form values on valid submission", async () => {
    render(LLMTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(vi.mocked(setLLMConfig)).toHaveBeenCalledWith({
      provider: "ollama",
      endpoint: defaultConfig.endpoint,
      model: defaultConfig.model,
      enabled: defaultConfig.enabled,
    });
  });
});

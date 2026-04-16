import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/svelte";
import { tick } from "svelte";
import type { Library } from "../../types";

vi.mock("../../stores/libraries.svelte", () => ({
  libraryStore: {
    add: vi.fn(),
  },
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "user-1",
      name: "Test User",
      email: "t@t.com",
      oidc_linked: false,
      is_admin: false,
    },
  },
}));

vi.mock("../../stores/onboarding.svelte", () => ({
  onboardingStore: {
    skip: vi.fn(),
    clearSkip: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  BookOpen: () => {},
  ChevronLeft: () => {},
  ChevronRight: () => {},
  FolderOpen: () => {},
  Plus: () => {},
  X: () => {},
}));

import FirstLibraryWizard from "./FirstLibraryWizard.svelte";
import { libraryStore } from "../../stores/libraries.svelte";
import { routerStore } from "../../stores/router.svelte";
import { onboardingStore } from "../../stores/onboarding.svelte";

const mockLibrary: Library = {
  id: "lib-new",
  name: "Fiction",
  paths: ["/books/fiction"],
  organization_type: "book_per_folder",
  monitored: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("FirstLibraryWizard – step 1 (name)", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the wizard with data-testid", () => {
    const { getByTestId } = render(FirstLibraryWizard);
    expect(getByTestId("first-library-wizard")).toBeInTheDocument();
  });

  it("renders step 1 heading 'Name your library'", () => {
    render(FirstLibraryWizard);
    expect(
      screen.getByRole("heading", { level: 1, name: /Name your library/i }),
    ).toBeInTheDocument();
  });

  it("renders the 'Skip for now' button on step 1", () => {
    render(FirstLibraryWizard);
    const skipButton = screen.getByRole("button", {
      name: /Skip.*setup.*now/i,
    });
    expect(skipButton).toBeInTheDocument();
    expect(skipButton).toHaveClass("text-ink-500");
    expect(skipButton).toHaveClass("dark:text-ink-400");
  });

  it("renders a 'Next' button on step 1", () => {
    render(FirstLibraryWizard);
    expect(screen.getByRole("button", { name: /Next/i })).toBeInTheDocument();
  });

  it("does not render a 'Back' button on step 1", () => {
    render(FirstLibraryWizard);
    expect(screen.queryByRole("button", { name: /Back/i })).toBeNull();
  });

  it("'Next' on step 1 with empty name shows a validation error", async () => {
    render(FirstLibraryWizard);
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
  });

  it("name input carries aria-invalid when validation fails", async () => {
    const { container } = render(FirstLibraryWizard);
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    const nameInput = container.querySelector("#wizard-lib-name");
    expect(nameInput).toHaveAttribute("aria-invalid", "true");
  });

  it("'Next' with a valid name advances to step 2", async () => {
    const { container } = render(FirstLibraryWizard);
    const nameInput = container.querySelector(
      "#wizard-lib-name",
    ) as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: "Fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    expect(
      screen.getByRole("heading", { level: 1, name: /Choose folders/i }),
    ).toBeInTheDocument();
  });
});

describe("FirstLibraryWizard – step 2 (paths)", () => {
  beforeEach(async () => {
    const { container } = render(FirstLibraryWizard);
    // Advance past step 1
    const nameInput = container.querySelector(
      "#wizard-lib-name",
    ) as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: "Fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the folder path input on step 2", () => {
    expect(
      screen.getByRole("textbox", { name: /Folder path/i }),
    ).toBeInTheDocument();
  });

  it("'Next' on step 2 with no paths shows a validation error", async () => {
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "At least one folder is required",
    );
  });

  it("'Back' returns to step 1", async () => {
    await fireEvent.click(screen.getByRole("button", { name: /Back/i }));
    await tick();
    expect(
      screen.getByRole("heading", { level: 1, name: /Name your library/i }),
    ).toBeInTheDocument();
  });

  it("'Next' with a valid path advances to step 3", async () => {
    const pathInput = screen.getByRole("textbox", { name: /Folder path/i });
    await fireEvent.input(pathInput, { target: { value: "/books/fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    expect(
      screen.getByRole("heading", { level: 1, name: /Configure options/i }),
    ).toBeInTheDocument();
  });
});

describe("FirstLibraryWizard – step 3 (options)", () => {
  async function advanceToStep3() {
    const { container } = render(FirstLibraryWizard);
    const nameInput = container.querySelector(
      "#wizard-lib-name",
    ) as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: "Fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    const pathInput = screen.getByRole("textbox", { name: /Folder path/i });
    await fireEvent.input(pathInput, { target: { value: "/books" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
  }

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the organization type select on step 3", async () => {
    await advanceToStep3();
    const { container } = { container: document.body };
    expect(container.querySelector("#wizard-org-type")).toBeInTheDocument();
  });

  it("renders the monitor toggle on step 3", async () => {
    await advanceToStep3();
    const { container } = { container: document.body };
    const toggle = container.querySelector("#wizard-monitored");
    expect(toggle).toBeInTheDocument();
    expect(toggle).toHaveAttribute("role", "switch");
  });

  it("renders organization helper text with accessible contrast classes on step 3", async () => {
    await advanceToStep3();
    const helperText = screen.getByText(
      "Determines how Biblioteka organizes books it imports into this library.",
    );
    expect(helperText).toHaveClass("text-ink-500");
    expect(helperText).toHaveClass("dark:text-ink-400");
  });

  it("'Next' on step 3 advances to step 4 (review)", async () => {
    await advanceToStep3();
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    expect(
      screen.getByRole("heading", { level: 1, name: /Review & create/i }),
    ).toBeInTheDocument();
  });
});

describe("FirstLibraryWizard – step 4 (review & create)", () => {
  async function advanceToStep4() {
    const { container } = render(FirstLibraryWizard);
    const nameInput = container.querySelector(
      "#wizard-lib-name",
    ) as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: "Fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    const pathInput = screen.getByRole("textbox", { name: /Folder path/i });
    await fireEvent.input(pathInput, { target: { value: "/books/fiction" } });
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
    await fireEvent.click(screen.getByRole("button", { name: /Next/i }));
    await tick();
  }

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the review step heading", async () => {
    await advanceToStep4();
    expect(
      screen.getByRole("heading", { level: 1, name: /Review & create/i }),
    ).toBeInTheDocument();
  });

  it("shows the library name in the review", async () => {
    await advanceToStep4();
    expect(screen.getByText("Fiction")).toBeInTheDocument();
  });

  it("shows the folder path in the review", async () => {
    await advanceToStep4();
    expect(screen.getByText("/books/fiction")).toBeInTheDocument();
  });

  it("shows the 'Create Library' button", async () => {
    await advanceToStep4();
    expect(
      screen.getByRole("button", { name: /Create Library/i }),
    ).toBeInTheDocument();
  });

  it("does not show a 'Next' button on step 4", async () => {
    await advanceToStep4();
    expect(screen.queryByRole("button", { name: /^Next$/i })).toBeNull();
  });

  it("calls libraryStore.add with the correct args on create", async () => {
    vi.mocked(libraryStore.add).mockResolvedValue(mockLibrary);
    await advanceToStep4();
    await fireEvent.click(
      screen.getByRole("button", { name: /Create Library/i }),
    );
    await tick();
    expect(libraryStore.add).toHaveBeenCalledWith({
      name: "Fiction",
      paths: ["/books/fiction"],
      organization_type: "book_per_folder",
      monitored: false,
    });
  });

  it("navigates to the new library after successful creation", async () => {
    vi.mocked(libraryStore.add).mockResolvedValue(mockLibrary);
    await advanceToStep4();
    await fireEvent.click(
      screen.getByRole("button", { name: /Create Library/i }),
    );
    await waitFor(() => {
      expect(routerStore.navigate).toHaveBeenCalledWith("libraries/lib-new");
    });
  });

  it("shows an error banner and stays on step 4 when creation fails", async () => {
    vi.mocked(libraryStore.add).mockRejectedValue(new Error("server error"));
    await advanceToStep4();
    await fireEvent.click(
      screen.getByRole("button", { name: /Create Library/i }),
    );
    await waitFor(() => {
      expect(screen.getByText("server error")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("heading", { level: 1, name: /Review & create/i }),
    ).toBeInTheDocument();
  });
});

describe("FirstLibraryWizard – skip", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("'Skip for now' calls onboardingStore.skip with the userId", async () => {
    render(FirstLibraryWizard);
    await fireEvent.click(
      screen.getByRole("button", { name: /Skip.*setup.*now/i }),
    );
    await tick();
    expect(onboardingStore.skip).toHaveBeenCalledWith("user-1");
  });

  it("'Skip for now' navigates to dashboard", async () => {
    render(FirstLibraryWizard);
    await fireEvent.click(
      screen.getByRole("button", { name: /Skip.*setup.*now/i }),
    );
    await tick();
    expect(routerStore.navigate).toHaveBeenCalledWith("dashboard");
  });
});

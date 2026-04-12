/**
 * popup.ts — Entry point for the browser extension popup.
 *
 * Responsibilities:
 *   1. Show the "Save to Biblioteka" button when settings are configured.
 *   2. Show a settings form when settings are missing (first run).
 *   3. Capture the current tab's URL on button click and report success/error.
 */

import type { Settings, Library } from "../types.js";
import { loadSettings, saveSettings, fetchLibraries, captureURL } from "../api.js";

const main = document.getElementById("main")!;

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

async function init(): Promise<void> {
  const settings = await loadSettings();

  if (!isConfigured(settings)) {
    renderSettings(settings);
  } else {
    renderCapture(settings as Settings);
  }
}

function isConfigured(s: Partial<Settings>): s is Settings {
  return !!(s.serverURL && s.apiKey);
}

// ---------------------------------------------------------------------------
// Capture view
// ---------------------------------------------------------------------------

async function renderCapture(settings: Settings): Promise<void> {
  // Get the active tab URL immediately so the UI renders quickly.
  let tabURL = "";
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    tabURL = tab?.url ?? "";
  } catch {
    // activeTab permission not yet granted; fall through with empty URL
  }

  // Fetch libraries in the background while the UI is building.
  let libraries: Library[] = [];
  let libraryError = "";
  try {
    libraries = await fetchLibraries(settings);
  } catch (err) {
    libraryError = err instanceof Error ? err.message : "Failed to load libraries";
  }

  const defaultLib = settings.defaultLibraryID;

  main.innerHTML = `
    <div class="capture-section">
      <p class="current-url" title="${escapeHTML(tabURL)}">${escapeHTML(tabURL || "(no URL)")}</p>
      ${
        libraryError
          ? `<div class="status error">${escapeHTML(libraryError)}</div>`
          : libraries.length === 0
            ? `<div class="status error">No libraries found. Create one in Biblioteka first.</div>`
            : `<div>
                <label for="library-select">Library</label>
                <select id="library-select">
                  ${libraries
                    .map(
                      (l) =>
                        `<option value="${escapeHTML(l.id)}"${l.id === defaultLib ? " selected" : ""}>${escapeHTML(l.name)}</option>`
                    )
                    .join("")}
                </select>
              </div>`
      }
      <button class="btn-primary" id="save-btn" ${libraries.length === 0 || libraryError ? "disabled" : ""}>
        Save to Biblioteka
      </button>
      <div id="status" role="status"></div>
      <div class="settings-link">
        <button id="open-settings">Settings</button>
      </div>
    </div>
  `;

  document.getElementById("open-settings")!.addEventListener("click", () => {
    renderSettings(settings);
  });

  const saveBtn = document.getElementById("save-btn") as HTMLButtonElement;
  const statusEl = document.getElementById("status")!;

  saveBtn.addEventListener("click", async () => {
    const librarySelect = document.getElementById("library-select") as HTMLSelectElement | null;
    const libraryID = librarySelect?.value ?? defaultLib;
    if (!libraryID) {
      showStatus(statusEl, "error", "No library selected.");
      return;
    }
    if (!tabURL) {
      showStatus(statusEl, "error", "Cannot determine current tab URL.");
      return;
    }

    saveBtn.disabled = true;
    showStatus(statusEl, "loading", "Saving…");

    try {
      await captureURL(settings, { url: tabURL, library_id: libraryID });
      // Persist the chosen library as the new default.
      await saveSettings({ ...settings, defaultLibraryID: libraryID });
      showStatus(statusEl, "success", "Saved! Processing in background…");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unknown error";
      showStatus(statusEl, "error", `Failed: ${msg}`);
      saveBtn.disabled = false;
    }
  });
}

// ---------------------------------------------------------------------------
// Settings view
// ---------------------------------------------------------------------------

function renderSettings(current: Partial<Settings>): void {
  main.innerHTML = `
    <div class="settings-section">
      <p class="settings-title">Configure Biblioteka</p>
      <div>
        <label for="server-url">Server URL</label>
        <input type="text" id="server-url"
          placeholder="https://biblioteka.example.com"
          value="${escapeHTML(current.serverURL ?? "")}" />
      </div>
      <div>
        <label for="api-key">API Key</label>
        <input type="text" id="api-key"
          placeholder="bib_…"
          value="${escapeHTML(current.apiKey ?? "")}" />
      </div>
      <button class="btn-primary" id="save-settings-btn">Save Settings</button>
      <div id="settings-status" role="status"></div>
      ${current.serverURL ? `<button class="btn-secondary" id="back-btn">← Back</button>` : ""}
    </div>
  `;

  const statusEl = document.getElementById("settings-status")!;

  document.getElementById("back-btn")?.addEventListener("click", () => {
    renderCapture(current as Settings);
  });

  document.getElementById("save-settings-btn")!.addEventListener("click", async () => {
    const serverURL = (document.getElementById("server-url") as HTMLInputElement).value.trim();
    const apiKey = (document.getElementById("api-key") as HTMLInputElement).value.trim();

    if (!serverURL) {
      showStatus(statusEl, "error", "Server URL is required.");
      return;
    }
    if (!apiKey) {
      showStatus(statusEl, "error", "API key is required.");
      return;
    }
    if (!serverURL.startsWith("http://") && !serverURL.startsWith("https://")) {
      showStatus(statusEl, "error", "Server URL must start with http:// or https://");
      return;
    }

    const updated: Settings = { serverURL, apiKey, defaultLibraryID: current.defaultLibraryID ?? "" };
    await saveSettings(updated);
    showStatus(statusEl, "success", "Settings saved.");
    setTimeout(() => renderCapture(updated), 800);
  });
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function showStatus(el: HTMLElement, type: "success" | "error" | "loading", message: string): void {
  el.className = `status ${type}`;
  el.textContent = message;
}

function escapeHTML(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

init().catch((err) => {
  main.innerHTML = `<div class="status error">Extension error: ${escapeHTML(String(err))}</div>`;
});

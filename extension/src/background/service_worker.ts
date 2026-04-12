/**
 * service_worker.ts — Background service worker for the Save to Biblioteka extension.
 *
 * Installs a context menu entry "Save to Biblioteka" on page link and frame
 * contexts so users can capture any open page with a right-click in addition
 * to using the popup.
 */

import type { Settings } from "../types.js";
import { loadSettings, captureURL } from "../api.js";

// ---------------------------------------------------------------------------
// Context menu
// ---------------------------------------------------------------------------

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus?.create({
    id: "save-to-biblioteka",
    title: "Save to Biblioteka",
    contexts: ["page", "frame"],
  });
});

chrome.contextMenus?.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== "save-to-biblioteka") return;

  const url = info.frameUrl ?? info.pageUrl ?? tab?.url;
  if (!url) return;

  const settings = await loadSettings();
  if (!isConfigured(settings)) {
    // Open the popup so the user can configure credentials.
    await chrome.action.openPopup?.();
    return;
  }

  const libraryID = settings.defaultLibraryID;
  if (!libraryID) {
    // No default library — ask the user to pick one via the popup.
    await chrome.action.openPopup?.();
    return;
  }

  try {
    await captureURL(settings as Settings, { url, library_id: libraryID });
    // Show a brief badge to confirm the capture was accepted.
    if (tab?.id != null) {
      await chrome.action.setBadgeText({ text: "✓", tabId: tab.id });
      await chrome.action.setBadgeBackgroundColor({ color: "#10b981" });
      setTimeout(() => {
        chrome.action.setBadgeText({ text: "", tabId: tab.id! });
      }, 3000);
    }
  } catch (err) {
    // Surface the error as an extension badge.
    if (tab?.id != null) {
      await chrome.action.setBadgeText({ text: "!", tabId: tab.id });
      await chrome.action.setBadgeBackgroundColor({ color: "#ef4444" });
      setTimeout(() => {
        chrome.action.setBadgeText({ text: "", tabId: tab.id! });
      }, 5000);
    }
    console.error("[Save to Biblioteka] capture failed:", err);
  }
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isConfigured(s: Partial<Settings>): s is Settings {
  return !!(s.serverURL && s.apiKey);
}

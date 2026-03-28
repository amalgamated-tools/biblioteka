/**
 * Copies text to the clipboard.
 *
 * Uses the modern async Clipboard API when available, and falls back to
 * `document.execCommand('copy')` for environments that don't support it.
 *
 * @throws {Error} when the copy operation fails using the available mechanism
 */
export async function copyToClipboard(text: string): Promise<void> {
  if (
    typeof navigator !== "undefined" &&
    navigator.clipboard &&
    typeof navigator.clipboard.writeText === "function"
  ) {
    await navigator.clipboard.writeText(text);
    return;
  }

  // Fallback for browsers/contexts without the async Clipboard API
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  try {
    const successful = document.execCommand("copy");
    if (!successful) {
      throw new Error("clipboard copy command was rejected");
    }
  } finally {
    document.body.removeChild(textarea);
  }

  if (!successful) {
    throw new Error("clipboard copy command was rejected");
  }
}

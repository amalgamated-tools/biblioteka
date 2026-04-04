/**
 * Manages the "copied" UI feedback state with an auto-reset timeout.
 *
 * Tracks which item (by ID) was most recently copied to the clipboard and
 * automatically clears that state after a configurable duration.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class CopyTimeoutState {
  copiedId: string | null = $state(null);

  private timeoutId: number | null = null;
  private readonly duration: number;

  constructor(duration = 2000) {
    this.duration = duration;
  }

  /**
   * Marks the given ID as copied and starts a timer to reset after `duration`
   * ms. Any previously running timer is cancelled first.
   */
  set(id: string): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
    }
    this.copiedId = id;
    this.timeoutId = window.setTimeout(() => {
      this.copiedId = null;
      this.timeoutId = null;
    }, this.duration);
  }

  /**
   * Cancels any pending timeout and resets `copiedId` to null.
   * Call this from `onDestroy` to prevent timer leaks.
   */
  clear(): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
      this.timeoutId = null;
    }
    this.copiedId = null;
  }
}

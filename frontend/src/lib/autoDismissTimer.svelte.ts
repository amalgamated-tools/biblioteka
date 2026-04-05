/**
 * Manages a temporary UI visibility state with an auto-dismiss timeout.
 *
 * Tracks whether a message (success, error, or informational) should be
 * visible and automatically clears that state after a configurable duration.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class AutoDismissTimer {
  visible = $state(false);

  private timeoutId: number | null = null;
  private readonly duration: number;

  constructor(duration = 3000) {
    this.duration = duration;
  }

  /**
   * Sets `visible` to true and starts a timer to reset it after `duration`
   * ms. Any previously running timer is cancelled first.
   */
  show(): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
    }
    this.visible = true;
    this.timeoutId = window.setTimeout(() => {
      this.visible = false;
      this.timeoutId = null;
    }, this.duration);
  }

  /**
   * Cancels any pending timeout and sets `visible` to false.
   * Call this from `onDestroy` to prevent timer leaks.
   */
  clear(): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
      this.timeoutId = null;
    }
    this.visible = false;
  }
}

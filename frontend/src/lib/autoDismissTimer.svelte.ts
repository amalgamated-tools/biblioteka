import { TimeoutState } from "./timeoutState.svelte";

/**
 * Manages a temporary UI visibility state with an auto-dismiss timeout.
 *
 * Tracks whether a message (success, error, or informational) should be
 * visible and automatically clears that state after a configurable duration.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class AutoDismissTimer extends TimeoutState<boolean> {
  constructor(duration = 3000) {
    super(false, duration);
  }

  /** Whether the associated UI element should currently be visible. */
  get visible(): boolean {
    return this.value;
  }

  /**
   * Sets `visible` to true and starts a timer to reset it after `duration`
   * ms. Any previously running timer is cancelled first.
   */
  show(): void {
    this.activate(true);
  }
}

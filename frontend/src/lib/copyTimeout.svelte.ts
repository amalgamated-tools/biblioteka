import { TimeoutState } from "./timeoutState.svelte";

/**
 * Manages the "copied" UI feedback state with an auto-reset timeout.
 *
 * Tracks which item (by ID) was most recently copied to the clipboard and
 * automatically clears that state after a configurable duration.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class CopyTimeoutState extends TimeoutState<string | null> {
  constructor(duration = 2000) {
    super(null, duration);
  }

  /** The ID of the most recently copied item, or null if none. */
  get copiedId(): string | null {
    return this.value;
  }

  /**
   * Marks the given ID as copied and starts a timer to reset after `duration`
   * ms. Any previously running timer is cancelled first.
   */
  set(id: string): void {
    this.activate(id);
  }
}

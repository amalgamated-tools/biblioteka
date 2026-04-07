/**
 * Generic timeout-based auto-resetting reactive state.
 *
 * Manages a reactive `value` that reverts to `idleValue` after `duration` ms.
 * Subclasses expose domain-specific getters and trigger methods on top of the
 * shared timer infrastructure.
 *
 * State is reactive via Svelte 5 `$state` runes.
 */
export class TimeoutState<T> {
  value: T = $state<T>(undefined as T);

  private timeoutId: number | null = null;
  private readonly idleValue: T;
  private readonly duration: number;

  constructor(idleValue: T, duration: number) {
    this.idleValue = idleValue;
    this.duration = duration;
    this.value = idleValue;
  }

  /**
   * Sets `value` to `activeValue` and starts a timer to revert to `idleValue`
   * after `duration` ms. Any previously running timer is cancelled first.
   */
  protected activate(activeValue: T): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
    }
    this.value = activeValue;
    this.timeoutId = window.setTimeout(() => {
      this.value = this.idleValue;
      this.timeoutId = null;
    }, this.duration);
  }

  /**
   * Cancels any pending timeout and resets `value` to `idleValue`.
   * Call this from `onDestroy` to prevent timer leaks.
   */
  clear(): void {
    if (this.timeoutId !== null) {
      clearTimeout(this.timeoutId);
      this.timeoutId = null;
    }
    this.value = this.idleValue;
  }
}

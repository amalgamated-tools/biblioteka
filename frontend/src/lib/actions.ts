import { tick } from "svelte";

/**
 * Svelte action that focuses the first button inside the given element
 * after the next microtask (tick), ensuring child components have rendered.
 */
export function autofocusFirstButton(node: HTMLElement) {
  tick().then(() => {
    const btn = node.querySelector<HTMLElement>("button");
    btn?.focus();
  });
}

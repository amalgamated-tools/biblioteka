<script lang="ts">
  interface Props {
    value?: string;
    type?: "text" | "email" | "password" | "url";
    id?: string;
    placeholder?: string;
    disabled?: boolean;
    autocomplete?: HTMLInputElement["autocomplete"];
    maxlength?: number;
    "aria-label"?: string;
    "aria-required"?: boolean;
    "aria-invalid"?: boolean;
    "aria-describedby"?: string;
    oninput?: (e: Event & { currentTarget: HTMLInputElement }) => void;
    class?: string;
  }

  let {
    value = $bindable(""),
    type = "text",
    id,
    placeholder,
    disabled,
    autocomplete,
    maxlength,
    "aria-label": ariaLabel,
    "aria-required": ariaRequired,
    "aria-invalid": ariaInvalid,
    "aria-describedby": ariaDescribedby,
    oninput,
    class: extraClass,
  }: Props = $props();
</script>

<input
  {type}
  {id}
  bind:value
  {placeholder}
  {disabled}
  {autocomplete}
  {maxlength}
  aria-label={ariaLabel}
  aria-required={ariaRequired}
  aria-invalid={ariaInvalid || undefined}
  aria-describedby={ariaDescribedby}
  {oninput}
  class="px-4 border border-ink-200 dark:border-ink-700 rounded-xl transition-all {disabled
    ? 'bg-ink-50 dark:bg-ink-800 text-ink-500 dark:text-ink-400 cursor-not-allowed'
    : 'focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 placeholder:text-ink-300 dark:placeholder:text-ink-500'} {extraClass ??
    ''}"
/>

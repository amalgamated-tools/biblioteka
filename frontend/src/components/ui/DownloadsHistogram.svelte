<script lang="ts">
  import type { MonthlyDownloads } from "../../types";

  interface Props {
    data: MonthlyDownloads[];
    /** Label shown above the chart */
    title?: string;
  }

  let { data, title = "Downloads per month" }: Props = $props();

  const maxCount = $derived(Math.max(...data.map((d) => d.count), 1));

  function barHeight(count: number): number {
    return Math.round((count / maxCount) * 100);
  }

  /**
   * Formats "YYYY-MM" as an abbreviated month label (e.g. "Jan", "Feb").
   * Falls back to the raw string if the date is invalid.
   */
  function monthLabel(yyyyMM: string): string {
    const [year, month] = yyyyMM.split("-");
    const d = new Date(Number(year), Number(month) - 1, 1);
    if (isNaN(d.getTime())) return yyyyMM;
    return d.toLocaleString("default", { month: "short" });
  }

  /** Returns "Month YYYY" for tooltip / aria-label. */
  function monthFull(yyyyMM: string): string {
    const [year, month] = yyyyMM.split("-");
    const d = new Date(Number(year), Number(month) - 1, 1);
    if (isNaN(d.getTime())) return yyyyMM;
    return d.toLocaleString("default", { month: "long", year: "numeric" });
  }

  const isEmpty = $derived(data.every((d) => d.count === 0));
</script>

<div class="w-full">
  <h3
    class="text-base font-semibold text-ink-700 dark:text-cream-200 mb-4"
    aria-label={title}
  >
    {title}
  </h3>

  {#if isEmpty}
    <p
      class="text-sm text-ink-400 dark:text-ink-400 text-center py-6"
      aria-live="polite"
    >
      No downloads recorded yet.
    </p>
  {:else}
    <div
      role="list"
      aria-label={title}
      class="flex items-end gap-1 h-32 w-full"
      data-testid="histogram-bars"
    >
      {#each data as item (item.month)}
        {@const h = barHeight(item.count)}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <div
          class="flex-1 flex flex-col items-center gap-1 group focus-within:outline-none"
          role="listitem"
          tabindex="0"
          aria-label="{monthFull(item.month)}: {item.count} {item.count === 1
            ? 'download'
            : 'downloads'}"
          title="{monthFull(item.month)}: {item.count}"
        >
          <span
            class="text-[10px] text-ink-400 dark:text-ink-400 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity leading-none"
            aria-hidden="true"
          >
            {item.count}
          </span>
          <div
            class="w-full rounded-t-sm bg-accent-400 dark:bg-accent-500 hover:bg-accent-500 dark:hover:bg-accent-400 group-focus-within:bg-accent-500 dark:group-focus-within:bg-accent-400 transition-colors"
            style="height: {h}%"
            role="presentation"
          ></div>
        </div>
      {/each}
    </div>

    <!-- Month labels -->
    <div class="flex gap-1 mt-1 w-full" aria-hidden="true">
      {#each data as item (item.month)}
        <div
          class="flex-1 text-center text-[10px] text-ink-400 dark:text-ink-400 truncate"
        >
          {monthLabel(item.month)}
        </div>
      {/each}
    </div>
  {/if}
</div>

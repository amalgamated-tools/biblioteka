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

  /** Returns "Month YYYY" for tooltip and screen-reader data table. */
  function monthFull(yyyyMM: string): string {
    const [year, month] = yyyyMM.split("-");
    const d = new Date(Number(year), Number(month) - 1, 1);
    if (isNaN(d.getTime())) return yyyyMM;
    return d.toLocaleString("default", { month: "long", year: "numeric" });
  }

  const isEmpty = $derived(data.every((d) => d.count === 0));
  const headingId = `histogram-title-${crypto.randomUUID()}`;
</script>

<div class="w-full">
  <h3
    id={headingId}
    class="text-base font-semibold text-ink-700 dark:text-cream-200 mb-4"
  >
    {title}
  </h3>

  {#if isEmpty}
    <p
      class="text-sm text-ink-500 dark:text-ink-300 text-center py-6"
      aria-live="polite"
    >
      No downloads recorded yet.
    </p>
  {:else}
    <div
      aria-hidden="true"
      class="flex items-end gap-1 h-32 w-full"
      data-testid="histogram-bars"
    >
      {#each data as item (item.month)}
        {@const h = barHeight(item.count)}
        <div
          class="flex-1 flex flex-col items-center justify-end gap-1 group h-full"
          title="{monthFull(item.month)}: {item.count}"
        >
          <span
            class="text-[10px] text-ink-600 dark:text-ink-200 opacity-0 group-hover:opacity-100 transition-opacity leading-none"
            aria-hidden="true"
          >
            {item.count}
          </span>
          <div
            class="w-full rounded-t-sm bg-accent-400 dark:bg-accent-500 hover:bg-accent-500 dark:hover:bg-accent-400 transition-colors"
            style:height="{h}%"
            role="presentation"
          ></div>
        </div>
      {/each}
    </div>

    <!-- Month labels -->
    <div class="flex gap-1 mt-1 w-full" aria-hidden="true">
      {#each data as item (item.month)}
        <div
          class="flex-1 text-center text-[10px] text-ink-600 dark:text-ink-300 truncate"
        >
          {monthLabel(item.month)}
        </div>
      {/each}
    </div>

    <table class="sr-only" data-testid="histogram-data-table">
      <caption>{title}</caption>
      <thead>
        <tr>
          <th scope="col">Month</th>
          <th scope="col">Downloads</th>
        </tr>
      </thead>
      <tbody>
        {#each data as item (item.month)}
          <tr>
            <th scope="row">{monthFull(item.month)}</th>
            <td>{item.count}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

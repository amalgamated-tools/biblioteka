<script lang="ts">
  import type { Tag } from "../../types";
  import * as api from "../../lib/api";
  import { tagStore } from "../../stores/tags.svelte";
  import { X, Tag as TagIcon, Plus, Loader } from "lucide-svelte";
  import AlertBanner from "../ui/AlertBanner.svelte";

  interface Props {
    bookId: string;
    disabled?: boolean;
  }

  let { bookId, disabled = false }: Props = $props();

  let assignedTags: Tag[] = $state.raw([]);
  let allTags: Tag[] = $state.raw([]);
  let loading = $state(true);
  let saving = $state(false);
  let error: string | null = $state(null);
  let searchText = $state("");
  let dropdownOpen = $state(false);
  let creatingTag = $state(false);
  let fetchSeq = 0;
  let saveSeq = 0;
  let activeIndex = $state(-1);

  $effect(() => {
    void loadData(bookId);
  });

  $effect(() => {
    void searchText;
    activeIndex = -1;
  });

  async function loadData(id: string) {
    const seq = ++fetchSeq;
    loading = true;
    error = null;
    assignedTags = [];
    allTags = [];
    searchText = "";
    dropdownOpen = false;
    activeIndex = -1;
    try {
      const [bookTags, tags] = await Promise.all([
        api.getBookTags(id),
        api.listTags(),
      ]);
      if (seq !== fetchSeq) return;
      assignedTags = bookTags;
      allTags = tags;
    } catch (e) {
      if (seq !== fetchSeq) return;
      error = e instanceof Error ? e.message : "Failed to load tags";
    } finally {
      if (seq === fetchSeq) loading = false;
    }
  }

  let assignedIds = $derived(new Set(assignedTags.map((t) => t.id)));

  let filteredTags = $derived(
    allTags.filter(
      (t) =>
        !assignedIds.has(t.id) &&
        t.name.toLowerCase().includes(searchText.toLowerCase().trim()),
    ),
  );

  let exactMatch = $derived(
    allTags.some(
      (t) => t.name.toLowerCase() === searchText.toLowerCase().trim(),
    ),
  );

  let showCreateOption = $derived(
    searchText.trim().length > 0 && !exactMatch && !creatingTag,
  );

  let activeOptionId: string | undefined = $derived(
    (() => {
      const total = filteredTags.length + (showCreateOption ? 1 : 0);
      if (activeIndex < 0 || activeIndex >= total) return undefined;
      if (activeIndex < filteredTags.length)
        return `book-tags-option-${filteredTags[activeIndex].id}`;
      return "book-tags-option-create";
    })(),
  );

  async function saveTags(tags: Tag[]) {
    const seq = ++saveSeq;
    const savedBookId = bookId;
    saving = true;
    error = null;
    try {
      const updated = await api.setBookTags(
        savedBookId,
        tags.map((t) => t.id),
      );
      if (seq !== saveSeq || bookId !== savedBookId) return;
      assignedTags = updated;
    } catch (e) {
      if (seq !== saveSeq || bookId !== savedBookId) return;
      error = e instanceof Error ? e.message : "Failed to save tags";
    } finally {
      if (seq === saveSeq) saving = false;
    }
  }

  async function addTag(tag: Tag) {
    const next = [...assignedTags, tag];
    searchText = "";
    dropdownOpen = false;
    await saveTags(next);
  }

  async function removeTag(tagId: string) {
    const next = assignedTags.filter((t) => t.id !== tagId);
    await saveTags(next);
  }

  async function createAndAddTag() {
    const name = searchText.trim();
    if (!name) return;
    creatingTag = true;
    error = null;
    try {
      const created = await tagStore.add({ name });
      allTags = [...allTags, created];
      await addTag(created);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to create tag";
    } finally {
      creatingTag = false;
    }
  }

  function handleInputKeydown(e: KeyboardEvent) {
    const total = filteredTags.length + (showCreateOption ? 1 : 0);
    if (e.key === "Escape") {
      dropdownOpen = false;
      searchText = "";
      activeIndex = -1;
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!dropdownOpen) {
        dropdownOpen = true;
        return;
      }
      activeIndex =
        total > 0 ? (activeIndex < total - 1 ? activeIndex + 1 : 0) : -1;
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (!dropdownOpen) {
        dropdownOpen = true;
        return;
      }
      activeIndex =
        total > 0 ? (activeIndex > 0 ? activeIndex - 1 : total - 1) : -1;
    } else if (e.key === "Home") {
      if (dropdownOpen && total > 0) {
        e.preventDefault();
        activeIndex = 0;
      }
    } else if (e.key === "End") {
      if (dropdownOpen && total > 0) {
        e.preventDefault();
        activeIndex = total - 1;
      }
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (activeIndex >= 0 && activeIndex < filteredTags.length) {
        void addTag(filteredTags[activeIndex]);
      } else if (activeIndex === filteredTags.length && showCreateOption) {
        void createAndAddTag();
      } else if (filteredTags.length === 1) {
        void addTag(filteredTags[0]);
      } else if (showCreateOption) {
        void createAndAddTag();
      }
    }
  }

  function handleInputFocus() {
    dropdownOpen = true;
  }

  function handleBlur(e: FocusEvent) {
    const related = e.relatedTarget as HTMLElement | null;
    if (!related?.closest("[data-tags-dropdown]")) {
      dropdownOpen = false;
      activeIndex = -1;
    }
  }
</script>

<div>
  <div class="flex items-center gap-2 mb-2">
    <TagIcon
      class="w-4 h-4 text-ink-500 dark:text-ink-400"
      aria-hidden="true"
    />
    <span class="text-sm font-medium text-ink-600 dark:text-ink-300">Tags</span>
    {#if saving}
      <Loader
        class="w-3.5 h-3.5 text-accent-500 animate-spin ml-1"
        aria-hidden="true"
      />
    {/if}
  </div>

  {#if error}
    <AlertBanner variant="error" class="mb-3">{error}</AlertBanner>
  {/if}

  {#if loading}
    <p role="status" class="text-sm text-ink-500 dark:text-ink-400">
      Loading tags...
    </p>
  {:else}
    <!-- Assigned tags chips -->
    <div class="flex flex-wrap gap-1.5 mb-3" aria-label="Assigned tags">
      {#each assignedTags as tag (tag.id)}
        <span
          class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-accent-100 text-accent-800 dark:bg-accent-800/30 dark:text-accent-300"
        >
          {tag.name}
          <button
            type="button"
            onclick={() => void removeTag(tag.id)}
            disabled={disabled || saving}
            aria-label={`Remove tag ${tag.name}`}
            class="ml-0.5 rounded-full hover:bg-accent-200 dark:hover:bg-accent-700/40 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <X class="w-3 h-3" aria-hidden="true" />
          </button>
        </span>
      {/each}
      {#if assignedTags.length === 0}
        <span class="text-xs text-ink-400 dark:text-ink-500 italic"
          >No tags assigned</span
        >
      {/if}
    </div>

    <!-- Tag search/add input -->
    <div class="relative" data-tags-dropdown>
      <div class="relative flex items-center">
        <Plus
          class="absolute left-3 w-3.5 h-3.5 text-ink-400 dark:text-ink-500 pointer-events-none"
          aria-hidden="true"
        />
        <input
          type="text"
          bind:value={searchText}
          placeholder="Add a tag…"
          disabled={disabled || saving}
          onfocus={handleInputFocus}
          onblur={handleBlur}
          onkeydown={handleInputKeydown}
          oninput={() => {
            dropdownOpen = true;
          }}
          aria-label="Search or add tags"
          aria-expanded={dropdownOpen}
          aria-autocomplete="list"
          role="combobox"
          aria-haspopup="listbox"
          aria-controls="book-tags-listbox"
          aria-activedescendant={activeOptionId}
          class="w-full pl-8 pr-4 py-2 text-sm border border-ink-200 dark:border-ink-700 rounded-xl bg-white dark:bg-ink-800 text-ink-900 dark:text-cream-100 placeholder:text-ink-400 dark:placeholder:text-ink-500 focus:ring-2 focus:ring-accent-500 focus:border-transparent focus-visible:outline-none transition-all disabled:opacity-50 disabled:cursor-not-allowed"
        />
      </div>

      {#if dropdownOpen && (filteredTags.length > 0 || showCreateOption)}
        <div
          id="book-tags-listbox"
          data-tags-dropdown
          role="listbox"
          aria-label="Available tags"
          class="absolute z-10 mt-1 w-full max-h-48 overflow-y-auto bg-white dark:bg-ink-800 border border-ink-200 dark:border-ink-700 rounded-xl shadow-lg py-1"
        >
          {#each filteredTags as tag, i (tag.id)}
            <button
              type="button"
              id="book-tags-option-{tag.id}"
              role="option"
              aria-selected={activeIndex === i}
              data-tags-dropdown
              onmousedown={(e) => e.preventDefault()}
              onclick={() => void addTag(tag)}
              disabled={disabled || saving}
              class="w-full text-left px-3 py-2 text-sm text-ink-700 dark:text-ink-200 hover:bg-accent-50 dark:hover:bg-accent-800/20 transition-colors disabled:opacity-50 {activeIndex ===
              i
                ? 'bg-accent-50 dark:bg-accent-800/20'
                : ''}"
            >
              {tag.name}
            </button>
          {/each}
          {#if showCreateOption}
            <button
              type="button"
              id="book-tags-option-create"
              role="option"
              aria-selected={activeIndex === filteredTags.length}
              data-tags-dropdown
              onmousedown={(e) => e.preventDefault()}
              onclick={() => void createAndAddTag()}
              disabled={disabled || saving || creatingTag}
              class="w-full text-left px-3 py-2 text-sm text-accent-600 dark:text-accent-400 hover:bg-accent-50 dark:hover:bg-accent-800/20 transition-colors disabled:opacity-50 flex items-center gap-1.5 {activeIndex ===
              filteredTags.length
                ? 'bg-accent-50 dark:bg-accent-800/20'
                : ''}"
            >
              <Plus class="w-3.5 h-3.5" aria-hidden="true" />
              Create "{searchText.trim()}"
            </button>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>

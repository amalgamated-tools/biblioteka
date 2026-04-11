<script lang="ts">
  import type { BookInput } from "../../types";
  import { routerStore } from "../../stores/router.svelte";
  import * as api from "../../lib/api";
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  export interface FormFields {
    title: string;
    description: string;
    publisher: string;
    language: string;
    publicationDate: string;
    isbn13: string;
    isbn10: string;
    asin: string;
    goodreadsId: string;
    hardcoverId: string;
    googleBooksId: string;
    coverImageUrl: string;
  }

  interface Props {
    bookId: string;
    fields: FormFields;
    saving?: boolean;
    hasPendingMetadata: boolean;
    onSaved: () => void;
  }

  let {
    bookId,
    fields = $bindable() as FormFields,
    saving = $bindable(false),
    hasPendingMetadata,
    onSaved,
  }: Props = $props();

  let formError: string | null = $state(null);

  async function handleSave(e: SubmitEvent) {
    e.preventDefault();
    if (!fields.title.trim()) {
      formError = "Title is required";
      return;
    }

    saving = true;
    formError = null;
    try {
      const input: BookInput = {
        title: fields.title.trim(),
        description: fields.description.trim() || null,
        publisher: fields.publisher.trim() || null,
        language: fields.language.trim() || null,
        publication_date: fields.publicationDate.trim() || null,
        isbn13: fields.isbn13.trim() || null,
        isbn10: fields.isbn10.trim() || null,
        asin: fields.asin.trim() || null,
        goodreads_id: fields.goodreadsId.trim() || null,
        hardcover_id: fields.hardcoverId.trim() || null,
        google_books_id: fields.googleBooksId.trim() || null,
        cover_image_url: fields.coverImageUrl.trim() || null,
      };
      await api.updateBook(bookId, input);
      // Clear pending metadata so it doesn't reappear on next visit.
      if (hasPendingMetadata) {
        try {
          await api.rejectMetadata(bookId);
        } catch {
          // Best effort — the book is already saved.
        }
      }
      onSaved();
    } catch (e) {
      formError = e instanceof Error ? e.message : "Failed to save book";
    } finally {
      saving = false;
    }
  }
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
>
  {#if formError}
    <AlertBanner variant="error" class="mb-4">{formError}</AlertBanner>
  {/if}

  <form onsubmit={handleSave} class="space-y-5">
    <p class="text-xs text-ink-500 dark:text-ink-300">
      Fields marked with <span aria-hidden="true">*</span><span class="sr-only"
        >an asterisk</span
      > are required.
    </p>
    <div>
      <label
        for="book-title"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
      >
        Title <span class="text-danger-600" aria-hidden="true">*</span>
      </label>
      <TextInput
        id="book-title"
        bind:value={fields.title}
        placeholder="Book title"
        class="w-full py-2.5"
        disabled={saving}
        aria-required={true}
      />
    </div>

    <div>
      <label
        for="book-description"
        class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
      >
        Description
      </label>
      <textarea
        id="book-description"
        bind:value={fields.description}
        placeholder="Book description"
        rows="3"
        class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 dark:placeholder-ink-500 transition-all resize-y"
        disabled={saving}
      ></textarea>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label
          for="book-publisher"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Publisher
        </label>
        <TextInput
          id="book-publisher"
          bind:value={fields.publisher}
          placeholder="Publisher"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-language"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Language
        </label>
        <TextInput
          id="book-language"
          bind:value={fields.language}
          placeholder="Language"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-pub-date"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Publication Date
        </label>
        <TextInput
          id="book-pub-date"
          bind:value={fields.publicationDate}
          placeholder="YYYY-MM-DD"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-isbn13"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          ISBN-13
        </label>
        <TextInput
          id="book-isbn13"
          bind:value={fields.isbn13}
          placeholder="ISBN-13"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-isbn10"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          ISBN-10
        </label>
        <TextInput
          id="book-isbn10"
          bind:value={fields.isbn10}
          placeholder="ISBN-10"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-asin"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          ASIN
        </label>
        <TextInput
          id="book-asin"
          bind:value={fields.asin}
          placeholder="ASIN"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-goodreads-id"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Goodreads ID
        </label>
        <TextInput
          id="book-goodreads-id"
          bind:value={fields.goodreadsId}
          placeholder="Goodreads ID"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-hardcover-id"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Hardcover ID
        </label>
        <TextInput
          id="book-hardcover-id"
          bind:value={fields.hardcoverId}
          placeholder="Hardcover ID"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-google-id"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Google Books ID
        </label>
        <TextInput
          id="book-google-id"
          bind:value={fields.googleBooksId}
          placeholder="Google Books ID"
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
      <div>
        <label
          for="book-cover-url"
          class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5"
        >
          Cover Image URL
        </label>
        <TextInput
          id="book-cover-url"
          bind:value={fields.coverImageUrl}
          placeholder="https://..."
          class="w-full py-2.5"
          disabled={saving}
        />
      </div>
    </div>

    <div class="flex items-center gap-3 pt-2">
      <Button
        type="submit"
        disabled={saving}
        class="px-5 py-2.5 text-sm active:scale-[0.98]"
      >
        {saving ? "Saving..." : "Save Changes"}
      </Button>
      <Button
        type="button"
        variant="secondary"
        onclick={() => routerStore.navigate(`books/${bookId}`)}
        disabled={saving}
        class="px-5 py-2.5 text-sm"
      >
        Cancel
      </Button>
    </div>
  </form>
</div>

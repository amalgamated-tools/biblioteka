<script lang="ts">
  import AlertBanner from "../ui/AlertBanner.svelte";
  import Button from "../ui/Button.svelte";
  import TextInput from "../ui/TextInput.svelte";

  interface Props {
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
    saving: boolean;
    formError: string | null;
    onsubmit: (e: SubmitEvent) => void;
    oncancel: () => void;
  }

  let {
    title = $bindable(""),
    description = $bindable(""),
    publisher = $bindable(""),
    language = $bindable(""),
    publicationDate = $bindable(""),
    isbn13 = $bindable(""),
    isbn10 = $bindable(""),
    asin = $bindable(""),
    goodreadsId = $bindable(""),
    hardcoverId = $bindable(""),
    googleBooksId = $bindable(""),
    coverImageUrl = $bindable(""),
    saving,
    formError,
    onsubmit,
    oncancel,
  }: Props = $props();
</script>

<div
  class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 p-6"
>
  {#if formError}
    <AlertBanner variant="error" class="mb-4">{formError}</AlertBanner>
  {/if}

  <form {onsubmit} class="space-y-5">
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
        bind:value={title}
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
        bind:value={description}
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
          bind:value={publisher}
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
          bind:value={language}
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
          bind:value={publicationDate}
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
          bind:value={isbn13}
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
          bind:value={isbn10}
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
          bind:value={asin}
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
          bind:value={goodreadsId}
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
          bind:value={hardcoverId}
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
          bind:value={googleBooksId}
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
          bind:value={coverImageUrl}
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
        onclick={oncancel}
        disabled={saving}
        class="px-5 py-2.5 text-sm"
      >
        Cancel
      </Button>
    </div>
  </form>
</div>

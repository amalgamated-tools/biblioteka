<script lang="ts">
  import { onMount } from "svelte";
  import { BookOpen, Send, X, Mail, FileText } from "lucide-svelte";
  import { bookStore } from "../stores/books.svelte";
  import * as api from "../lib/api";
  import type { Book, BookFile } from "../types";

  let error: string | null = $state(null);

  // Send-via-email modal state
  let sendModal: {
    book: Book;
    files: BookFile[];
    selectedFileId: string;
    email: string;
    sending: boolean;
    sent: boolean;
    error: string | null;
  } | null = $state(null);

  onMount(async () => {
    if (!bookStore.loaded) {
      try {
        await bookStore.load();
      } catch (e) {
        error = e instanceof Error ? e.message : "Failed to load books";
      }
    }
  });

  async function openSendModal(bookId: string) {
    error = null;
    try {
      const [book, files] = await Promise.all([
        api.getBook(bookId),
        api.listBookFiles(bookId),
      ]);
      sendModal = {
        book,
        files,
        selectedFileId: files.length > 0 ? files[0].id : "",
        email: "",
        sending: false,
        sent: false,
        error: null,
      };
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load book details";
    }
  }

  function closeSendModal() {
    sendModal = null;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (sendModal && e.key === "Escape") {
      closeSendModal();
    }
  }

  async function handleSend() {
    if (!sendModal) return;
    sendModal.error = null;

    const emailTrimmed = sendModal.email.trim();
    if (!emailTrimmed) {
      sendModal.error = "Email address is required";
      return;
    }
    if (!emailTrimmed.includes("@")) {
      sendModal.error = "Please enter a valid email address";
      return;
    }
    if (!sendModal.selectedFileId) {
      sendModal.error = "Please select a file to send";
      return;
    }

    sendModal.sending = true;
    try {
      await api.sendBookFile(sendModal.selectedFileId, emailTrimmed);
      sendModal.sent = true;
    } catch (e) {
      sendModal.error = e instanceof Error ? e.message : "Failed to send email";
    } finally {
      sendModal.sending = false;
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div>
  <div class="flex items-center gap-3 mb-8">
    <div class="w-10 h-10 bg-accent-100 dark:bg-accent-800/20 rounded-xl flex items-center justify-center">
      <BookOpen class="w-5 h-5 text-accent-600 dark:text-accent-400" />
    </div>
    <h1 class="text-3xl font-display font-bold text-ink-900 dark:text-cream-100">All Books</h1>
  </div>

  {#if error}
    <div class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm mb-4 animate-scale-in">
      {error}
    </div>
  {/if}

  {#if bookStore.loading}
    <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800">
      <div class="text-center py-8">
        <p class="text-ink-400 dark:text-ink-400">Loading books…</p>
      </div>
    </div>
  {:else if bookStore.books.length === 0}
    <div class="bg-white dark:bg-ink-900 rounded-2xl p-8 shadow-sm border border-ink-100 dark:border-ink-800 animate-fade-in">
      <div class="text-center py-8">
        <BookOpen class="w-12 h-12 text-ink-200 dark:text-ink-700 mx-auto mb-4" />
        <p class="text-ink-400 dark:text-ink-400 text-lg">
          No books yet.
        </p>
        <p class="text-ink-300 dark:text-ink-500 text-sm mt-1">
          Books will appear here once they are added to your libraries.
        </p>
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each bookStore.books as book (book.id)}
        <div class="bg-white dark:bg-ink-900 rounded-2xl shadow-sm border border-ink-100 dark:border-ink-800 overflow-hidden hover:shadow-md transition-shadow group">
          {#if book.cover_image_url}
            <div class="aspect-[2/3] bg-ink-100 dark:bg-ink-800">
              <img
                src={book.cover_image_url}
                alt={book.title}
                class="w-full h-full object-cover"
              />
            </div>
          {:else}
            <div class="aspect-[2/3] bg-ink-100 dark:bg-ink-800 flex items-center justify-center">
              <BookOpen class="w-10 h-10 text-ink-300 dark:text-ink-600" />
            </div>
          {/if}
          <div class="p-3">
            <h3 class="font-medium text-sm text-ink-900 dark:text-cream-100 truncate" title={book.title}>
              {book.title}
            </h3>
            {#if book.publisher}
              <p class="text-xs text-ink-400 dark:text-ink-500 truncate mt-0.5">{book.publisher}</p>
            {/if}
            <button
              onclick={() => openSendModal(book.id)}
              class="mt-2 w-full inline-flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-accent-600 dark:text-accent-400 border border-accent-200 dark:border-accent-800/50 rounded-lg hover:bg-accent-50 dark:hover:bg-accent-800/20 transition-colors"
              title="Send via email"
            >
              <Send class="w-3.5 h-3.5" />
              Send via Email
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Send via Email modal -->
{#if sendModal}
  <!-- Backdrop: press Escape or click outside to close -->
  <div
    role="presentation"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-ink-950/60 backdrop-blur-sm animate-fade-in"
    onclick={(e) => { if (e.target === e.currentTarget) closeSendModal(); }}
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="send-modal-title"
      class="bg-white dark:bg-ink-900 rounded-2xl shadow-xl border border-ink-100 dark:border-ink-800 w-full max-w-md animate-scale-in"
    >
      <div class="flex items-center justify-between px-6 py-4 border-b border-ink-100 dark:border-ink-800">
        <div class="flex items-center gap-2">
          <Mail class="w-5 h-5 text-accent-600 dark:text-accent-400" />
          <h2 id="send-modal-title" class="text-lg font-display font-bold text-ink-900 dark:text-cream-100">Send via Email</h2>
        </div>
        <button
          type="button"
          onclick={closeSendModal}
          class="text-ink-300 hover:text-ink-500 dark:hover:text-ink-200 transition-colors"
          aria-label="Close"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <div class="px-6 py-5 space-y-4">
        <div>
          <p class="text-sm font-medium text-ink-700 dark:text-ink-300 truncate" title={sendModal.book.title}>
            {sendModal.book.title}
          </p>
        </div>

        {#if sendModal.sent}
          <div class="flex items-center gap-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800/40 text-green-700 dark:text-green-400 px-4 py-3 rounded-xl text-sm">
            <Send class="w-4 h-4 flex-shrink-0" />
            Email sent successfully to {sendModal.email}!
          </div>
          <button
            type="button"
            onclick={closeSendModal}
            class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 text-ink-600 dark:text-ink-300 rounded-xl hover:bg-ink-50 dark:hover:bg-ink-800 transition-all text-sm font-medium"
          >
            Close
          </button>
        {:else}
          {#if sendModal.files.length === 0}
            <div class="text-sm text-ink-400 dark:text-ink-500 py-2">
              This book has no files available to send.
            </div>
          {:else}
            {#if sendModal.files.length > 1}
              <div>
                <label for="send-file-select" class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5">
                  File
                </label>
                <div class="space-y-1.5">
                  {#each sendModal.files as file (file.id)}
                    <label class="flex items-center gap-2.5 cursor-pointer group/file">
                      <input
                        type="radio"
                        name="send-file"
                        value={file.id}
                        bind:group={sendModal.selectedFileId}
                        class="accent-accent-600"
                      />
                      <FileText class="w-3.5 h-3.5 text-ink-400 flex-shrink-0" />
                      <span class="text-sm text-ink-700 dark:text-ink-300 truncate">
                        {file.file_name}
                        <span class="text-xs text-ink-400 uppercase ml-1">{file.file_type}</span>
                      </span>
                    </label>
                  {/each}
                </div>
              </div>
            {:else}
              <div class="flex items-center gap-2 text-sm text-ink-600 dark:text-ink-300">
                <FileText class="w-4 h-4 text-ink-400 flex-shrink-0" />
                <span class="truncate">{sendModal.files[0].file_name}</span>
                <span class="text-xs text-ink-400 uppercase">{sendModal.files[0].file_type}</span>
              </div>
            {/if}

            <div>
              <label for="send-email" class="block text-sm font-medium text-ink-600 dark:text-ink-300 mb-1.5">
                Recipient email address
              </label>
              <input
                id="send-email"
                type="email"
                bind:value={sendModal.email}
                placeholder="e.g. user@kindle.com"
                class="w-full px-4 py-2.5 border border-ink-200 dark:border-ink-700 rounded-xl focus:ring-2 focus:ring-accent-500 focus:border-transparent outline-none dark:bg-ink-800 dark:text-cream-100 transition-all placeholder:text-ink-300 dark:placeholder:text-ink-500 text-sm"
                disabled={sendModal.sending}
                onkeydown={(e) => { if (e.key === "Enter") handleSend(); }}
              />
            </div>

            {#if sendModal.error}
              <div class="bg-danger-50 dark:bg-danger-700/10 border border-danger-600/20 dark:border-danger-700/30 text-danger-700 dark:text-red-400 px-4 py-3 rounded-xl text-sm animate-scale-in">
                {sendModal.error}
              </div>
            {/if}

            <div class="flex gap-3 pt-1">
              <button
                type="button"
                onclick={handleSend}
                disabled={sendModal.sending || !sendModal.selectedFileId}
                class="flex-1 inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-gradient-to-r from-accent-600 to-accent-700 text-white rounded-xl hover:from-accent-700 hover:to-accent-800 transition-all text-sm font-semibold disabled:opacity-50 shadow-md shadow-accent-600/20 active:scale-[0.98]"
              >
                <Send class="w-4 h-4" />
                {sendModal.sending ? "Sending…" : "Send"}
              </button>
              <button
                type="button"
                onclick={closeSendModal}
                disabled={sendModal.sending}
                class="px-5 py-2.5 border border-ink-200 dark:border-ink-700 text-ink-600 dark:text-ink-300 rounded-xl hover:bg-ink-50 dark:hover:bg-ink-800 transition-all text-sm font-medium"
              >
                Cancel
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
{/if}


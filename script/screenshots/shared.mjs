import { chromium } from '@playwright/test';
import { fileURLToPath } from 'url';
import { mkdir } from 'fs/promises';
import path from 'path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const screenshotsDir = path.join(__dirname, '..', '..', 'screenshots');
const demoBooksDir = path.join(__dirname, '.demo-books');
const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
const DEMO_NAME = process.env.DEMO_NAME || 'Demo';
const DEMO_EMAIL = process.env.DEMO_EMAIL || 'demo@veverka.net';
const DEMO_PASSWORD = process.env.DEMO_PASSWORD || 'password123';
const NONADMIN_NAME = process.env.NONADMIN_NAME || 'Regular User';
const NONADMIN_EMAIL = process.env.NONADMIN_EMAIL || 'nonadmin@veverka.net';
const NONADMIN_PASSWORD = process.env.NONADMIN_PASSWORD || 'password123';
const DEFAULT_TIMEOUT_MS = Number(process.env.SCREENSHOT_TIMEOUT_MS || 5000);
const NAVIGATION_TIMEOUT_MS = Number(process.env.SCREENSHOT_NAVIGATION_TIMEOUT_MS || 8000);
export const AUTH_ERROR_TEST_ID = 'auth-error';

const VIEWPORTS = {
    desktop: { width: 1440, height: 900 },
    mobile: { width: 375, height: 812 },
};

export function getAuthErrorBanner(page) {
    return page.getByTestId(AUTH_ERROR_TEST_ID);
}

function setTheme(page, theme) {
    return page.evaluate((value) => {
        localStorage.setItem('biblioteka_theme', value);
        document.documentElement.dataset.theme = value;
    }, theme);
}

async function openAuthPage(page) {
    await page.goto(`${BASE_URL}/`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.waitForSelector('button#login-tab');
}

async function openSignupForm(page) {
    await openAuthPage(page);
    await page.locator('button#signup-tab').click();
    await page.waitForSelector('input#signup-name');
    await page.waitForFunction(() => {
        const panel = document.querySelector('#signup-panel:not([hidden])');
        const btn = panel && panel.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
}

async function waitForSignupTab(page) {
    await page.waitForFunction(() => {
        const signupBtn = document.querySelector('button#signup-tab');
        const loginBtn = document.querySelector('button#login-tab');
        return (
            signupBtn &&
            loginBtn &&
            signupBtn.classList.contains('bg-white') &&
            !loginBtn.classList.contains('bg-white')
        );
    });
    await page.waitForTimeout(150);
}

async function logoutIfNeeded(page) {
    // Clear browser cookies to remove the HttpOnly session cookie
    await page.context().clearCookies();
    await page.goto(`${BASE_URL}/`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.waitForSelector('button#login-tab', { timeout: NAVIGATION_TIMEOUT_MS });
}

async function loginAsDemo(page) {
    await openAuthPage(page);
    await page.locator('input#login-email').fill(DEMO_EMAIL);
    await page.locator('input#login-password').fill(DEMO_PASSWORD);
    await page.locator('#login-panel button[type="submit"]').click();
    await page.getByRole('button', { name: 'Logout', exact: true }).waitFor({ state: 'visible' });
}

async function openSettingsPage(page) {
    await page.goto(`${BASE_URL}/#settings/account`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Settings', exact: true }).waitFor({ state: 'visible' });
    await page.locator('input#email-display').waitFor({ state: 'visible' });
}

async function openSettingsTab(page, tabHash, headingName) {
    await page.goto(`${BASE_URL}/#settings/${tabHash}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: headingName, exact: true }).waitFor({ state: 'visible' });
}

async function openBooksPage(page) {
    await page.goto(`${BASE_URL}/#books`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // If no libraries exist, the app redirects to dashboard; wait for either
    await Promise.any([
        page.getByRole('heading', { name: 'All Books', exact: true }).waitFor({ state: 'visible' }),
        page.getByRole('heading', { name: 'Dashboard', exact: true }).waitFor({ state: 'visible' }),
    ]);
    // Wait for the book list to finish loading (if we're on the books page)
    const booksHeading = page.getByRole('heading', { name: 'All Books', exact: true });
    if (await booksHeading.count()) {
        await page.waitForFunction(() => !document.body.innerText.includes('Loading books...'));
    }
}

async function openBooksPageWithSearch(page, query) {
    const encoded = encodeURIComponent(query);
    await page.goto(`${BASE_URL}/#books?query=${encoded}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'All Books', exact: true }).waitFor({ state: 'visible' });
    await page.waitForFunction(() => !document.body.innerText.includes('Loading books...'));
}

async function openBookDetailPage(page, bookId) {
    await page.goto(`${BASE_URL}/#books/${bookId}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for the book title h1 (aria-busy is set while loading, removed once the book is fetched)
    await page.waitForFunction(() => {
        const h1 = document.querySelector('main h1');
        return h1 && !h1.hasAttribute('aria-busy') && h1.textContent.trim().length > 0;
    });
}

async function openBookEditPage(page, bookId) {
    await page.goto(`${BASE_URL}/#books/${bookId}/edit`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Edit Book', exact: true }).waitFor({ state: 'visible' });
    await page.waitForFunction(() => !document.body.innerText.includes('Loading book...'));
}

async function openMyLibraryPage(page) {
    await page.goto(`${BASE_URL}/#my-library`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'My Library', exact: true }).waitFor({ state: 'visible' });
}

async function openLibrariesPage(page) {
    await page.goto(`${BASE_URL}/#libraries`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for either the empty-state CTA, or the "no libraries" redirect CTA, or
    // the populated placeholder shown when libraries exist
    await Promise.any([
        page.getByRole('button', { name: 'Add A Library' }).waitFor({ state: 'visible' }),
        page.getByText('Select a library from the sidebar').waitFor({ state: 'visible' }),
        page.getByText('Select a library from the sidebar or create a new one.').waitFor({ state: 'visible' }),
    ]);
}

async function openLibrarySetupPage(page) {
    await page.goto(`${BASE_URL}/#libraries/setup`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByTestId('first-library-wizard').waitFor({ state: 'visible' });
}

async function openLibraryNewPage(page) {
    await page.goto(`${BASE_URL}/#libraries/new`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for the name input in the library creation form
    await page.locator('input#lib-name').waitFor({ state: 'visible' });
}

async function openLibraryViewPage(page, libraryId) {
    await page.goto(`${BASE_URL}/#libraries/${libraryId}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for the book list to settle (scanning indicator or empty state)
    await page.waitForFunction(() => !document.body.innerText.includes('Loading books...'));
}

async function openLibraryEditPage(page, libraryId) {
    await page.goto(`${BASE_URL}/#libraries/edit/${libraryId}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Edit Library', exact: true }).waitFor({ state: 'visible' });
}

async function openReadingListsPage(page) {
    await page.goto(`${BASE_URL}/#reading-lists`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Reading Lists', exact: true }).waitFor({ state: 'visible' });
}

async function openReadingListDetailPage(page, listId) {
    await page.goto(`${BASE_URL}/#reading-lists/${listId}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for the list name h1 to appear (only rendered once the store has loaded)
    await page.waitForFunction(() => {
        const h1 = document.querySelector('main h1');
        return h1 && h1.textContent.trim().length > 0;
    });
}

async function openGroupsPage(page) {
    await page.goto(`${BASE_URL}/#groups`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Reading Groups', exact: true }).waitFor({ state: 'visible' });
}

async function openGroupDetailPage(page, groupId) {
    await page.goto(`${BASE_URL}/#groups/${groupId}`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    // Wait for the group name h1 to appear (only rendered once the store has loaded)
    await page.waitForFunction(() => {
        const h1 = document.querySelector('main h1');
        return h1 && h1.textContent.trim().length > 0;
    });
}

async function openTagsPage(page) {
    await page.goto(`${BASE_URL}/#tags`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Tags', exact: true }).waitFor({ state: 'visible' });
}

async function openNotFoundPage(page) {
    await page.goto(`${BASE_URL}/#this-route-does-not-exist`, {
        waitUntil: 'networkidle',
        timeout: NAVIGATION_TIMEOUT_MS,
    });
    await page.getByRole('heading', { name: 'Page Not Found', exact: true }).waitFor({ state: 'visible' });
}

async function ensureAccount(page, name, email, password) {
    console.log(`Ensuring account exists for ${email}...`);
    await openSignupForm(page);
    await page.locator('input#signup-name').fill(name);
    await page.locator('input#signup-email').fill(email);
    await page.locator('input#signup-password').fill(password);
    await page.locator('#signup-panel button[type="submit"]').click();

    const logoutButton = page.getByRole('button', { name: 'Logout', exact: true });
    const errorBanner = getAuthErrorBanner(page);

    await Promise.any([
        logoutButton.waitFor({ state: 'visible' }),
        errorBanner.waitFor({ state: 'visible' }),
    ]);

    if (await errorBanner.count()) {
        const errorText = (await errorBanner.first().innerText()).trim();
        if (!/taken|exist|already/i.test(errorText)) {
            throw new Error(`Failed to create account for ${email}: ${errorText}`);
        }
        console.log(`Account already exists for ${email}, continuing...`);
    }

    await logoutIfNeeded(page);
}

async function ensureDemoAccount(page) {
    await ensureAccount(page, DEMO_NAME, DEMO_EMAIL, DEMO_PASSWORD);
}

async function ensureNonadminAccount(page) {
    await ensureAccount(page, NONADMIN_NAME, NONADMIN_EMAIL, NONADMIN_PASSWORD);
}

async function loginAsNonadmin(page) {
    await openAuthPage(page);
    await page.locator('input#login-email').fill(NONADMIN_EMAIL);
    await page.locator('input#login-password').fill(NONADMIN_PASSWORD);
    await page.locator('#login-panel button[type="submit"]').click();
    await page.getByRole('button', { name: 'Logout', exact: true }).waitFor({ state: 'visible' });
}

/**
 * Injects shared fetch helpers into the page context as window globals.
 * Must be called before any page.evaluate that uses __apiGet/__apiPost/__apiPut.
 * Re-inject after any navigation that clears the page context.
 */
async function injectApiHelpers(page) {
    await page.evaluate(() => {
        window.__apiGet = async function (urlPath) {
            const res = await fetch(urlPath);
            if (!res.ok) throw new Error(`GET ${urlPath} failed (${res.status})`);
            return res.json();
        };
        window.__apiPost = async function (urlPath, body) {
            const res = await fetch(urlPath, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            if (!res.ok) {
                const text = await res.text();
                throw new Error(`POST ${urlPath} failed (${res.status}): ${text}`);
            }
            if (res.status === 204) return null;
            return res.json();
        };
        window.__apiPut = async function (urlPath, body) {
            const res = await fetch(urlPath, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            if (!res.ok) {
                const text = await res.text();
                throw new Error(`PUT ${urlPath} failed (${res.status}): ${text}`);
            }
            return res.json();
        };
    });
}

/**
 * Seeds a library via the API. Idempotent: skips creation if "Demo Library"
 * already exists. Returns the library ID.
 * Must be called while a user is logged in.
 */
async function seedLibrary(page, demoBooksPath) {
    await injectApiHelpers(page);
    return page.evaluate(async (demoBooksPath) => {
        const libs = await window.__apiGet('/api/libraries');
        const existing = libs.find((l) => l.name === 'Demo Library');
        if (existing) return { libraryId: existing.id };

        const library = await window.__apiPost('/api/libraries', {
            name: 'Demo Library',
            paths: [demoBooksPath],
            organization_type: 'book_per_folder',
            monitored: false,
        });
        return { libraryId: library.id };
    }, demoBooksPath);
}

/**
 * Seeds books, authors, tags, reading lists, and a reading group via the API.
 * Idempotent: skips all creation if books already exist and returns existing IDs.
 * Returns entity IDs for use in navigation.
 * Must be called while a user is logged in.
 */
async function seedBooksAndMore(page) {
    await injectApiHelpers(page);
    return page.evaluate(async () => {
        const booksResult = await window.__apiGet('/api/books');
        if (booksResult.total > 0) {
            const lists = await window.__apiGet('/api/reading-lists');
            const groups = await window.__apiGet('/api/groups');
            return {
                bookIds: booksResult.books.slice(0, 5).map((b) => b.id),
                listIds: lists.slice(0, 2).map((l) => l.id),
                groupIds: groups.slice(0, 1).map((g) => g.id),
            };
        }

        const bookInputs = [
            {
                title: 'Dune',
                description: 'Epic science fiction set on the desert planet Arrakis.',
                publication_date: '1965-08-01',
                publisher: 'Chilton Books',
                language: 'en',
            },
            {
                title: 'The Name of the Wind',
                description: "The legendary story of Kvothe from his own point of view.",
                publication_date: '2007-03-27',
                publisher: 'DAW Books',
                language: 'en',
            },
            {
                title: 'Neuromancer',
                description: 'A seminal cyberpunk novel set in a sprawling future.',
                publication_date: '1984-07-01',
                publisher: 'Ace Books',
                language: 'en',
            },
            {
                title: 'The Left Hand of Darkness',
                description: "A lone human ambassador's journey on a genderless planet.",
                publication_date: '1969-03-01',
                publisher: 'Ace Books',
                language: 'en',
            },
            {
                title: 'Foundation',
                description: 'The fall and rise of a Galactic Empire over millennia.',
                publication_date: '1951-05-01',
                publisher: 'Gnome Press',
                language: 'en',
            },
        ];
        const books = await Promise.all(bookInputs.map((b) => window.__apiPost('/api/books', b)));
        const bookIds = books.map((b) => b.id);

        const authorInputs = [
            { name: 'Frank Herbert' },
            { name: 'Patrick Rothfuss' },
            { name: 'William Gibson' },
        ];
        const authors = await Promise.all(authorInputs.map((a) => window.__apiPost('/api/authors', a)));
        const authorIds = authors.map((a) => a.id);

        await window.__apiPut(`/api/books/${bookIds[0]}/authors`, { author_ids: [authorIds[0]] });
        await window.__apiPut(`/api/books/${bookIds[1]}/authors`, { author_ids: [authorIds[1]] });
        await window.__apiPut(`/api/books/${bookIds[2]}/authors`, { author_ids: [authorIds[2]] });

        const tagInputs = [{ name: 'sci-fi' }, { name: 'fantasy' }, { name: 'classic' }];
        const tags = await Promise.all(tagInputs.map((t) => window.__apiPost('/api/tags', t)));
        const tagIds = tags.map((t) => t.id);

        await window.__apiPut(`/api/books/${bookIds[0]}/tags`, { tag_ids: [tagIds[0], tagIds[2]] });
        await window.__apiPut(`/api/books/${bookIds[1]}/tags`, { tag_ids: [tagIds[1]] });
        await window.__apiPut(`/api/books/${bookIds[2]}/tags`, { tag_ids: [tagIds[0]] });
        await window.__apiPut(`/api/books/${bookIds[4]}/tags`, { tag_ids: [tagIds[0], tagIds[2]] });

        const list1 = await window.__apiPost('/api/reading-lists', {
            name: 'To Read',
            description: 'Books I want to read next',
        });
        const list2 = await window.__apiPost('/api/reading-lists', {
            name: 'Favorites',
            description: 'All-time favorites',
        });
        const listIds = [list1.id, list2.id];

        await window.__apiPost(`/api/reading-lists/${listIds[0]}/books`, { book_id: bookIds[1] });
        await window.__apiPost(`/api/reading-lists/${listIds[0]}/books`, { book_id: bookIds[2] });
        await window.__apiPost(`/api/reading-lists/${listIds[1]}/books`, { book_id: bookIds[0] });

        const group1 = await window.__apiPost('/api/groups', {
            name: 'Sci-Fi Book Club',
            description: 'A group for science fiction enthusiasts',
        });
        const groupIds = [group1.id];

        return { bookIds, authorIds, tagIds, listIds, groupIds };
    });
}

function buildFilename(screen, variantName) {
    return `${screen}-${variantName}.png`;
}

function getViewport(mobile) {
    return mobile ? VIEWPORTS.mobile : VIEWPORTS.desktop;
}

export async function runVariant({ theme, mobile }) {
    const variantName = mobile ? `${theme}-mobile` : theme;
    await mkdir(screenshotsDir, { recursive: true });
    await mkdir(demoBooksDir, { recursive: true });

    const browser = await chromium.launch();
    const context = await browser.newContext({
        viewport: getViewport(mobile),
        deviceScaleFactor: 2,
    });

    try {
        const page = await context.newPage();
        page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
        page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);

        // ── Phase 0: Unauthenticated ──────────────────────────────────────────

        console.log(`Capturing login (${variantName})...`);
        await openAuthPage(page);
        await setTheme(page, theme);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.waitForSelector('button#login-tab');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('login', variantName)) });

        console.log(`Capturing signup (${variantName})...`);
        await setTheme(page, theme);
        await openSignupForm(page);
        await waitForSignupTab(page);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('signup', variantName)) });

        // ── Phase 1: Post-login empty states (no seeded data) ─────────────────

        await ensureDemoAccount(page);
        await loginAsDemo(page);
        await setTheme(page, theme);

        console.log(`Capturing dashboard-empty (${variantName})...`);
        await page.goto(`${BASE_URL}/#dashboard`, {
            waitUntil: 'networkidle',
            timeout: NAVIGATION_TIMEOUT_MS,
        });
        await page.getByRole('heading', { name: 'Dashboard', exact: true }).waitFor({ state: 'visible' });
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('dashboard-empty', variantName)) });

        console.log(`Capturing dashboard-empty-skipped (${variantName})...`);
        // Inject the onboarding-skipped flag for this user so the alternate
        // empty-state card ("You skipped setup") is shown instead of the wizard CTA.
        await page.evaluate(async () => {
            const res = await fetch('/api/auth/me');
            const user = await res.json();
            localStorage.setItem(`biblioteka_onboarding_skipped_${user.id}`, '1');
        });
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.getByRole('heading', { name: 'Dashboard', exact: true }).waitFor({ state: 'visible' });
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('dashboard-empty-skipped', variantName)) });
        // Clear the flag so subsequent screenshots start from a clean state
        await page.evaluate(async () => {
            const res = await fetch('/api/auth/me');
            const user = await res.json();
            localStorage.removeItem(`biblioteka_onboarding_skipped_${user.id}`);
        });

        console.log(`Capturing libraries-empty (${variantName})...`);
        await openLibrariesPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('libraries-empty', variantName)) });

        console.log(`Capturing libraries-setup (${variantName})...`);
        await openLibrarySetupPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('libraries-setup', variantName)) });

        console.log(`Capturing libraries-new (${variantName})...`);
        await openLibraryNewPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('libraries-new', variantName)) });

        console.log(`Capturing reading-lists-empty (${variantName})...`);
        await openReadingListsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('reading-lists-empty', variantName)) });

        console.log(`Capturing groups-empty (${variantName})...`);
        await openGroupsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('groups-empty', variantName)) });

        console.log(`Capturing tags-empty (${variantName})...`);
        await openTagsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('tags-empty', variantName)) });

        console.log(`Capturing not-found (${variantName})...`);
        await openNotFoundPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('not-found', variantName)) });

        // ── Phase 2a: Seed library only (needed before books-empty) ───────────

        console.log(`Seeding library (${variantName})...`);
        const { libraryId } = await seedLibrary(page, demoBooksDir);

        // ── Phase 3a: books-empty (library exists, no scanned books yet) ──────

        console.log(`Capturing books-empty (${variantName})...`);
        await openBooksPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('books-empty', variantName)) });

        // ── Phase 2b: Seed books, authors, tags, lists, group ────────────────

        console.log(`Seeding books and more (${variantName})...`);
        const { bookIds, listIds, groupIds } = await seedBooksAndMore(page);

        // ── Phase 3b: Populated screenshots ───────────────────────────────────

        console.log(`Capturing dashboard-populated (${variantName})...`);
        await page.goto(`${BASE_URL}/#dashboard`, {
            waitUntil: 'networkidle',
            timeout: NAVIGATION_TIMEOUT_MS,
        });
        await page.getByRole('heading', { name: 'Dashboard', exact: true }).waitFor({ state: 'visible' });
        await setTheme(page, theme);
        // Brief pause so async dashboard stats (counts, recommendations) can render
        await page.waitForTimeout(1500);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('dashboard-populated', variantName)) });

        console.log(`Capturing books-list (${variantName})...`);
        await openBooksPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('books-list', variantName)) });

        console.log(`Capturing books-search (${variantName})...`);
        await openBooksPageWithSearch(page, 'dune');
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('books-search', variantName)) });

        console.log(`Capturing book-detail (${variantName})...`);
        await openBookDetailPage(page, bookIds[0]);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('book-detail', variantName)) });

        console.log(`Capturing book-edit (${variantName})...`);
        await openBookEditPage(page, bookIds[0]);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('book-edit', variantName)) });

        console.log(`Capturing libraries-list (${variantName})...`);
        await openLibrariesPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('libraries-list', variantName)) });

        console.log(`Capturing library-view (${variantName})...`);
        await openLibraryViewPage(page, libraryId);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('library-view', variantName)) });

        console.log(`Capturing library-edit (${variantName})...`);
        await openLibraryEditPage(page, libraryId);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('library-edit', variantName)) });

        console.log(`Capturing reading-lists-list (${variantName})...`);
        await openReadingListsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('reading-lists-list', variantName)) });

        console.log(`Capturing reading-list-detail (${variantName})...`);
        await openReadingListDetailPage(page, listIds[0]);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('reading-list-detail', variantName)) });

        console.log(`Capturing groups-list (${variantName})...`);
        await openGroupsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('groups-list', variantName)) });

        console.log(`Capturing group-detail (${variantName})...`);
        await openGroupDetailPage(page, groupIds[0]);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('group-detail', variantName)) });

        console.log(`Capturing tags-list (${variantName})...`);
        await openTagsPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('tags-list', variantName)) });

        console.log(`Capturing my-library (${variantName})...`);
        await openMyLibraryPage(page);
        await setTheme(page, theme);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('my-library', variantName)) });

        // ── Phase 4: Admin settings ───────────────────────────────────────────

        console.log(`Capturing settings (${variantName})...`);
        await openSettingsPage(page);
        await setTheme(page, theme);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.getByRole('heading', { name: 'Settings', exact: true }).waitFor({ state: 'visible' });
        await page.locator('input#email-display').waitFor({ state: 'visible' });
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings', variantName)) });

        console.log(`Capturing settings API keys (${variantName})...`);
        await openSettingsTab(page, 'api-keys', 'API Keys');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-api-keys', variantName)) });

        console.log(`Capturing settings Kobo sync (${variantName})...`);
        await openSettingsTab(page, 'kobo', 'Kobo Sync');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-kobo', variantName)) });

        console.log(`Capturing settings preferences (${variantName})...`);
        await openSettingsTab(page, 'preferences', 'Display Preferences');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-preferences', variantName)) });

        console.log(`Capturing settings OIDC (${variantName})...`);
        await openSettingsTab(page, 'oidc', 'OIDC / Single Sign-On');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-oidc', variantName)) });

        console.log(`Capturing settings SMTP (${variantName})...`);
        await openSettingsTab(page, 'smtp', 'Email / SMTP Configuration');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-smtp', variantName)) });

        console.log(`Capturing settings Watch Folder (${variantName})...`);
        await openSettingsTab(page, 'watch-folder', 'Watch Folder');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-watch-folder', variantName)) });

        console.log(`Capturing settings Calibre Import (${variantName})...`);
        await openSettingsTab(page, 'calibre-import', 'Import from Calibre');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-calibre-import', variantName)) });

        console.log(`Capturing settings Search Index (${variantName})...`);
        await openSettingsTab(page, 'search-index', 'Search Index');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-search-index', variantName)) });

        console.log(`Capturing settings users (${variantName})...`);
        await openSettingsTab(page, 'users', 'User Management');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-users', variantName)) });

        console.log(`Logging out admin for ${variantName}...`);
        await logoutIfNeeded(page);

        // ── Phase 5: Non-admin settings ───────────────────────────────────────

        console.log(`Ensuring non-admin account (${variantName})...`);
        await ensureNonadminAccount(page);
        await loginAsNonadmin(page);
        await setTheme(page, theme);

        console.log(`Capturing non-admin settings (${variantName})...`);
        await openSettingsPage(page);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.getByRole('heading', { name: 'Settings', exact: true }).waitFor({ state: 'visible' });
        await page.locator('input#email-display').waitFor({ state: 'visible' });
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-nonadmin', variantName)) });

        console.log(`Capturing non-admin settings API keys (${variantName})...`);
        await openSettingsTab(page, 'api-keys', 'API Keys');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-nonadmin-api-keys', variantName)) });

        console.log(`Capturing non-admin settings Kobo sync (${variantName})...`);
        await openSettingsTab(page, 'kobo', 'Kobo Sync');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-nonadmin-kobo', variantName)) });

        console.log(`Capturing non-admin settings preferences (${variantName})...`);
        await openSettingsTab(page, 'preferences', 'Display Preferences');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-nonadmin-preferences', variantName)) });

        console.log(`Logging out non-admin for ${variantName}...`);
        await logoutIfNeeded(page);

        console.log(`Saved screenshots for ${variantName} in screenshots/.`);
    } finally {
        await context.close();
        await browser.close();
    }
}

import { chromium } from '@playwright/test';
import { fileURLToPath } from 'url';
import { mkdir } from 'fs/promises';
import path from 'path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const screenshotsDir = path.join(__dirname, '..', 'screenshots');
const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

function setTheme(page, theme) {
    return page.evaluate((t) => {
        localStorage.setItem('biblioteka_theme', t);
        document.documentElement.dataset.theme = t;
    }, theme);
}

async function main() {
    await mkdir(screenshotsDir, { recursive: true });

    const browser = await chromium.launch();
    const context = await browser.newContext({
        viewport: { width: 1440, height: 900 },
        deviceScaleFactor: 2,
    });

    const page = await context.newPage();

    // ── 1. Login — light mode ─────────────────────
    console.log('📸 Login (light)...');
    await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
    // the button has been updated to id="login-btn"
    await page.waitForSelector('button#login-btn');
    await setTheme(page, 'light');
    await page.screenshot({ path: path.join(screenshotsDir, 'login-light.png') });

    // ── 2. Login — dark mode ──────────────────────
    console.log('📸 Login (dark)...');
    await setTheme(page, 'dark');
    // reload the page to ensure dark theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForSelector('button#login-btn');
    await page.screenshot({ path: path.join(screenshotsDir, 'login-dark.png') });

    // ── 3. Sign up — light mode ───────────────────
    console.log('📸 Sign up (light)...');
    await setTheme(page, 'light');
    // reload the page to ensure light theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
    await page.waitForSelector('input#name');
    // wait for the submit button text to update to "Create Account"
    await page.waitForFunction(() => {
        const btn = document.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
    // make sure that the signup-btn is highlight and the login button is not
    await page.waitForFunction(() => {
        const signupBtn = document.querySelector('button#signup-btn');
        const loginBtn = document.querySelector('button#login-btn');
        return (
            signupBtn &&
            loginBtn &&
            signupBtn.classList.contains('bg-white') &&
            !loginBtn.classList.contains('bg-white')
        );
    });
    // sleep for a short time to ensure all styles are applied (e.g. the active state of the buttons)
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(screenshotsDir, 'signup-light.png') });

    // ── 4. Sign up — dark mode ────────────────────
    console.log('📸 Sign up (dark)...');
    await setTheme(page, 'dark');
    // reload the page to ensure dark theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
    await page.waitForSelector('input#name');
    // wait for the submit button text to update to "Create Account"
    await page.waitForFunction(() => {
        const btn = document.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
    // make sure that the signup-btn is highlight and the login button is not
    await page.waitForFunction(() => {
        const signupBtn = document.querySelector('button#signup-btn');
        const loginBtn = document.querySelector('button#login-btn');
        return (
            signupBtn &&
            loginBtn &&
            signupBtn.classList.contains('bg-white') &&
            !loginBtn.classList.contains('bg-white')
        );
    });
    // sleep for a short time to ensure all styles are applied (e.g. the active state of the buttons)
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(screenshotsDir, 'signup-dark.png') });

    await page.setViewportSize({ width: 375, height: 812 });

    // ── 1. Login — light mode ─────────────────────
    console.log('📸 Login (light)...');
    await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
    // the button has been updated to id="login-btn"
    await page.waitForSelector('button#login-btn');
    await setTheme(page, 'light');
    await page.screenshot({ path: path.join(screenshotsDir, 'login-mobile-light.png') });

    // ── 2. Login — dark mode ──────────────────────
    console.log('📸 Login (dark)...');
    await setTheme(page, 'dark');
    // reload the page to ensure dark theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForSelector('button#login-btn');
    await page.screenshot({ path: path.join(screenshotsDir, 'login-mobile-dark.png') });

    // ── 3. Sign up — light mode ───────────────────
    console.log('📸 Sign up (light)...');
    await setTheme(page, 'light');
    // reload the page to ensure light theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
    await page.waitForSelector('input#name');
    // wait for the submit button text to update to "Create Account"
    await page.waitForFunction(() => {
        const btn = document.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
    // make sure that the signup-btn is highlight and the login button is not
    await page.waitForFunction(() => {
        const signupBtn = document.querySelector('button#signup-btn');
        const loginBtn = document.querySelector('button#login-btn');
        return (
            signupBtn &&
            loginBtn &&
            signupBtn.classList.contains('bg-white') &&
            !loginBtn.classList.contains('bg-white')
        );
    });
    // sleep for a short time to ensure all styles are applied (e.g. the active state of the buttons)
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(screenshotsDir, 'signup-mobile-light.png') });

    // ── 4. Sign up — dark mode ────────────────────
    console.log('📸 Sign up (dark)...');
    await setTheme(page, 'dark');
    // reload the page to ensure dark theme is applied to all elements (e.g. the theme toggle button)
    await page.reload({ waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
    await page.waitForSelector('input#name');
    // wait for the submit button text to update to "Create Account"
    await page.waitForFunction(() => {
        const btn = document.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
    // make sure that the signup-btn is highlight and the login button is not
    await page.waitForFunction(() => {
        const signupBtn = document.querySelector('button#signup-btn');
        const loginBtn = document.querySelector('button#login-btn');
        return (
            signupBtn &&
            loginBtn &&
            signupBtn.classList.contains('bg-white') &&
            !loginBtn.classList.contains('bg-white')
        );
    });
    // sleep for a short time to ensure all styles are applied (e.g. the active state of the buttons)
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(screenshotsDir, 'signup-mobile-dark.png') });

    await browser.close();
    console.log('✅ All screenshots saved to screenshots/');
}

main().catch((err) => {
    console.error('Screenshot script failed:', err);
    process.exit(1);
});

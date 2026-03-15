import { chromium } from '@playwright/test';
import { fileURLToPath } from 'url';
import { mkdir } from 'fs/promises';
import path from 'path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const screenshotsDir = path.join(__dirname, '..', '..', 'screenshots');
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
    await page.waitForSelector('button#login-btn');
}

async function openSignupForm(page) {
    await openAuthPage(page);
    await page.getByRole('button', { name: 'Sign Up', exact: true }).click();
    await page.waitForSelector('input#name');
    await page.waitForFunction(() => {
        const btn = document.querySelector('button[type="submit"]');
        return btn && btn.textContent.trim() === 'Create Account';
    });
}

async function waitForSignupTab(page) {
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
    await page.waitForTimeout(150);
}

async function logoutIfNeeded(page) {
    await page.locator('button#logout-button').click().catch(() => { });
}

async function loginAsDemo(page) {
    await openAuthPage(page);
    await page.locator('input#email').fill(DEMO_EMAIL);
    await page.locator('input#password').fill(DEMO_PASSWORD);
    await page.locator('button[type="submit"]').click();
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

async function openSettingsTab(page, tabName, headingName) {
    await page.getByRole('button', { name: tabName, exact: true }).click();
    await page.getByRole('heading', { name: headingName, exact: true }).waitFor({ state: 'visible' });
}

async function ensureAccount(page, name, email, password) {
    console.log(`Ensuring account exists for ${email}...`);
    await openSignupForm(page);
    await page.locator('input#name').fill(name);
    await page.locator('input#email').fill(email);
    await page.locator('input#password').fill(password);
    await page.locator('button[type="submit"]').click();

    const logoutButton = page.getByRole('button', { name: 'Logout', exact: true });
    const errorBanner = getAuthErrorBanner(page);

    await Promise.race([
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
    await page.locator('input#email').fill(NONADMIN_EMAIL);
    await page.locator('input#password').fill(NONADMIN_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.getByRole('button', { name: 'Logout', exact: true }).waitFor({ state: 'visible' });
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

    const browser = await chromium.launch();
    const context = await browser.newContext({
        viewport: getViewport(mobile),
        deviceScaleFactor: 2,
    });

    try {
        const page = await context.newPage();
        page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
        page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);

        console.log(`Capturing login (${variantName})...`);
        await openAuthPage(page);
        await setTheme(page, theme);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.waitForSelector('button#login-btn');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('login', variantName)) });

        console.log(`Capturing signup (${variantName})...`);
        await setTheme(page, theme);
        await openSignupForm(page);
        await waitForSignupTab(page);
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('signup', variantName)) });

        console.log(`Capturing dashboard (${variantName})...`);
        await ensureDemoAccount(page);
        await loginAsDemo(page);
        await setTheme(page, theme);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        // wait for the logout-button to ensure the dashboard is fully loaded before taking a screenshot
        // await page.getByRole('button', { name: 'Logout', exact: true }).waitFor({ state: 'visible' });
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('dashboard', variantName)) });

        console.log(`Capturing settings (${variantName})...`);
        await openSettingsPage(page);
        await setTheme(page, theme);
        await page.reload({ waitUntil: 'networkidle', timeout: NAVIGATION_TIMEOUT_MS });
        await page.getByRole('heading', { name: 'Settings', exact: true }).waitFor({ state: 'visible' });
        await page.locator('input#email-display').waitFor({ state: 'visible' });
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings', variantName)) });

        console.log(`Capturing settings OIDC (${variantName})...`);
        await openSettingsTab(page, 'OIDC / SSO', 'OIDC / Single Sign-On');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-oidc', variantName)) });

        console.log(`Capturing settings users (${variantName})...`);
        await openSettingsTab(page, 'Users', 'User Management');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-users', variantName)) });

        console.log(`Capturing settings preferences (${variantName})...`);
        await openSettingsTab(page, 'Preferences', 'Display Preferences');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-preferences', variantName)) });

        console.log(`Logging out admin for ${variantName}...`);
        await logoutIfNeeded(page);

        // Non-admin settings screenshots
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

        console.log(`Capturing non-admin settings preferences (${variantName})...`);
        await openSettingsTab(page, 'Preferences', 'Display Preferences');
        await page.screenshot({ path: path.join(screenshotsDir, buildFilename('settings-nonadmin-preferences', variantName)) });

        console.log(`Logging out non-admin for ${variantName}...`);
        await logoutIfNeeded(page);

        console.log(`Saved screenshots for ${variantName} in screenshots/.`);
    } finally {
        await context.close();
        await browser.close();
    }
}

import { runVariant } from './screenshots/shared.mjs';

const variants = [
    { theme: 'light', mobile: false },
    { theme: 'dark', mobile: false },
    { theme: 'light', mobile: true },
    { theme: 'dark', mobile: true },
];

async function main() {
    for (const variant of variants) {
        await runVariant(variant);
    }
}

main().catch((err) => {
    console.error('Screenshot script failed:', err);
    process.exit(1);
});

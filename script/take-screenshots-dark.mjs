import { runVariant } from './screenshots/shared.mjs';

runVariant({ theme: 'dark', mobile: false }).catch((err) => {
    console.error('Screenshot script failed:', err);
    process.exit(1);
});

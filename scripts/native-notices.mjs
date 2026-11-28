import {appendFile,readFile} from 'node:fs/promises';
import {collectNotices} from './collect-notices.mjs';
await collectNotices(process.argv[2],[{CGO_ENABLED:process.env.CGO_ENABLED}], './desktop');
await collectNotices(process.argv[3],[{CGO_ENABLED:process.env.CGO_ENABLED}], './cmd/companion');

await appendFile(process.argv[2], '\n\n## Lucide SVG icons (0.456.0)\n'+await readFile('desktop/frontend/ui/icons/LICENSE','utf8'));

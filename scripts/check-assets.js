import { readdirSync, readFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { join } from 'node:path';

const root = 'internal/web/static';
const allowed = new Set(['app.js', 'connection.js', 'model.js', 'preferences.js', 'app.css', 'themes.css']);
const files = readdirSync(root).filter(name => /\.(js|css)$/.test(name));
const unexpected = files.filter(name => !allowed.has(name));
if (unexpected.length) throw new Error(`Untracked runtime dependencies: ${unexpected.join(', ')}`);
const sizes = files.map(name => {
  const data = readFileSync(join(root, name));
  return { name, rawBytes: data.length, gzipBytes: gzipSync(data, { level: 9 }).length };
});
const totals = sizes.reduce((a, s) => ({ rawBytes: a.rawBytes + s.rawBytes, gzipBytes: a.gzipBytes + s.gzipBytes }), { rawBytes: 0, gzipBytes: 0 });
if (totals.rawBytes > 128 * 1024 || totals.gzipBytes > 32 * 1024) throw new Error('Reader asset budget exceeded: review before raising the budget');
const css = files.filter(name => name.endsWith('.css')).map(name => readFileSync(join(root, name), 'utf8')).join('\n');
if (/transition\s*:\s*(all|[.\d]+m?s\b)/.test(css)) throw new Error('Avoid all/implicit-all transitions');
const report = { note: 'Per-file gzip sizes are measurements, not a claim that the Go server compresses responses.', files: sizes, totals, budget: { rawBytes: 131072, gzipBytes: 32768 } };
mkdirSync('artifacts', { recursive: true });
writeFileSync('artifacts/asset-sizes.json', JSON.stringify(report, null, 2));
console.log(JSON.stringify(report, null, 2));

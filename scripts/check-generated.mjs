// Compare bytes rather than Git state so this also works before a first commit.
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve, relative } from 'node:path';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';

const root = fileURLToPath(new URL('../tree-sitter-qlik/', import.meta.url));
const src = resolve(root, 'src');
function snapshot(dir = src, out = new Map()) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const name = resolve(dir, entry.name);
    if (entry.isDirectory()) snapshot(name, out);
    else out.set(relative(src, name), createHash('sha256').update(readFileSync(name)).digest('hex'));
  }
  return out;
}
const before = snapshot();
const result = spawnSync(resolve(root, 'node_modules/.bin/tree-sitter'), ['generate'], { cwd: root, stdio: 'inherit' });
if (result.error) { console.error(result.error.message); process.exit(1); }
if (result.status !== 0) process.exit(result.status ?? 1);
const after = snapshot();
const changed = [...new Set([...before.keys(), ...after.keys()])].filter(name => before.get(name) !== after.get(name));
if (changed.length) {
  console.error(`Generated files changed: ${changed.join(', ')}. Review and include regenerated artifacts.`);
  process.exit(1);
}
console.log('Generated parser artifacts are reproducible.');

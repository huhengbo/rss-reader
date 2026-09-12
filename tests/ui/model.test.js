import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parseSnapshot, newItemCount, contentKey, safeLink } from '../../internal/web/static/model.js';
import '../../internal/web/static/preferences.js';

const sample = () => ({ version: 1, revision: 'r1', autoUpdatePush: 1, listHeight: 600, title: 'Reader', description: '', sources: [
  { id: 'one', title: 'One', link: 'https://example.com', status: 'ready', items: [{ id: 'a', title: 'Alpha', link: 'https://example.com/a' }] },
  { id: 'two', title: 'Two', link: 'https://example.com', status: 'empty', items: [] }
] });

test('snapshot preserves same-homepage sources and authoritative empty lists', () => {
  assert.equal(parseSnapshot(sample()).sources.length, 2);
  const s = sample(); s.sources = [];
  assert.deepEqual(parseSnapshot(s).sources, []);
});
test('reject malformed, duplicate and unsupported snapshots', () => {
  for (const value of [null, {}, { ...sample(), version: 2 }, { ...sample(), sources: [sample().sources[0], sample().sources[0]] }]) {
    assert.throws(() => parseSnapshot(value));
  }
});
test('new content count ignores time/status changes and detects later items', () => {
  const a = sample(), b = sample(); b.sources[0].lastSuccessAt = '2026-01-01T00:00:00Z';
  assert.equal(contentKey(a), contentKey(b));
  b.sources[0].items.push({ id: 'b', title: 'Beta', link: 'https://example.com/b' });
  assert.equal(newItemCount(a, b), 1);
  assert.notEqual(contentKey(a), contentKey(b));
});
test('only absolute credential-free http(s) links are interactive', () => {
  for (const value of ['javascript:alert(1)', 'data:text/html,test', '//evil.example', 'https://user:pass@example.com', null]) assert.equal(safeLink(value), '');
  assert.equal(safeLink('https://example.com/?id=2'), 'https://example.com/?id=2');
});
test('preferences tolerate corrupt values and unavailable storage', () => {
  const p = globalThis.ReaderPreferences;
  assert.deepEqual(p.normalize({ skin: 'invalid', mode: 'invalid', density: 'invalid' }), { skin: 'slate', mode: 'system', density: 'comfortable' });
  assert.equal(p.read({ getItem() { return '{bad'; } }).skin, 'slate');
  assert.equal(p.read({ getItem() { throw new Error('blocked'); } }).mode, 'system');
  assert.doesNotThrow(() => p.save({ setItem() { throw new Error('quota'); } }, {}));
});

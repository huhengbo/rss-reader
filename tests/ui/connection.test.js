import { test } from 'node:test';
import assert from 'node:assert/strict';
import { ReaderConnection } from '../../internal/web/static/connection.js';

function environment(t) {
  const originals = new Map();
  function set(name, value) {
    originals.set(name, Object.getOwnPropertyDescriptor(globalThis, name));
    Object.defineProperty(globalThis, name, { value, configurable: true, writable: true });
  }
  const timers = new Map(), sockets = [];
  let next = 0;
  class Target extends EventTarget {
    listeners = new Set();
    addEventListener(name, listener) { this.listeners.add(listener); super.addEventListener(name, listener); }
    removeEventListener(name, listener) { this.listeners.delete(listener); super.removeEventListener(name, listener); }
  }
  const win = new Target(), doc = new Target();
  win.location = { href: 'https://reader.example/' };
  doc.hidden = false;
  set('window', win); set('document', doc); set('navigator', { onLine: true });
  set('setTimeout', (callback, delay) => { const id = ++next; timers.set(id, { callback, delay }); return id; });
  set('clearTimeout', id => timers.delete(id));
  class Socket {
    static OPEN = 1;
    readyState = 1;
    constructor(url) { this.url = String(url); sockets.push(this); }
    close(code = 1000) { this.readyState = 3; this.onclose?.({ code }); }
    message(value) { this.onmessage?.({ data: JSON.stringify(value) }); }
  }
  set('WebSocket', Socket);
  let requestSignal;
  set('fetch', (_, { signal }) => {
    requestSignal = signal;
    return new Promise((resolve, reject) => signal.addEventListener('abort', () => reject(new Error('aborted')), { once: true }));
  });
  t.after(() => { for (const [name, descriptor] of originals) { if (descriptor) Object.defineProperty(globalThis, name, descriptor); else delete globalThis[name]; } });
  return { win, doc, timers, sockets, signal: () => requestSignal, fire() { const [id, timer] = timers.entries().next().value; timers.delete(id); timer.callback(); } };
}

const snapshot = { version: 1, revision: 'r1', title: 'Reader', description: '', autoUpdatePush: 1, sources: [] };

test('failed stream schedules one bounded retry and stop removes all resources', t => {
  const e = environment(t);
  const states = [];
  const reader = new ReaderConnection({ mode: 1, onSnapshot() {}, onStatus: s => states.push(s) });
  reader.start(); reader.start();
  assert.equal(e.sockets.length, 1);
  assert.equal(e.sockets[0].url, 'wss://reader.example/ws?v=1');
  e.sockets[0].close(1006); e.sockets[0].close(1006);
  assert.equal(e.timers.size, 1);
  assert.ok([...e.timers.values()][0].delay >= 1000 && [...e.timers.values()][0].delay <= 1250);
  e.fire();
  assert.equal(e.sockets.length, 2);
  e.sockets[1].close(1006);
  assert.equal(e.timers.size, 1);
  reader.stop(); reader.stop();
  assert.equal(e.timers.size, 0);
  assert.equal(e.win.listeners.size + e.doc.listeners.size, 0);
  e.win.dispatchEvent(new Event('online'));
  assert.equal(e.sockets.length, 2);
  assert.ok(states.includes('reconnecting'));
});

test('complete one-shot snapshot closes normally without retry', t => {
  const e = environment(t);
  let latest;
  const reader = new ReaderConnection({ mode: 0, onSnapshot: s => { latest = s; }, onStatus() {} });
  reader.start();
  e.sockets[0].message({ ...snapshot, autoUpdatePush: 0 });
  e.sockets[0].close(1000);
  reader.connect();
  assert.equal(latest.version, 1);
  assert.equal(e.sockets.length, 1);
  assert.equal(e.timers.size, 0);
  reader.stop();
});

test('a new stream snapshot aborts stale HTTP response; hidden page cancels its work', async t => {
  const e = environment(t);
  const values = [];
  const reader = new ReaderConnection({ mode: 1, onSnapshot: s => values.push(s.revision), onStatus() {} });
  reader.start();
  const first = reader.refresh();
  const firstSignal = e.signal();
  e.sockets[0].message(snapshot);
  await first;
  assert.equal(firstSignal.aborted, true);
  assert.deepEqual(values, ['r1']);
  const next = reader.refresh();
  e.doc.hidden = true;
  e.doc.dispatchEvent(new Event('visibilitychange'));
  await next;
  assert.equal(e.signal().aborted, true);
  assert.equal(reader.socket, null);
  assert.equal(e.timers.size, 0);
  reader.stop();
  assert.equal(e.win.listeners.size + e.doc.listeners.size, 0);
});

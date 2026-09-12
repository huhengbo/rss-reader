import { parseSnapshot } from './model.js';

// Owns exactly one WebSocket, retry timer and cancellable snapshot request.
export class ReaderConnection {
  constructor({ onSnapshot, onStatus, mode }) {
    this.onSnapshot = onSnapshot;
    this.onStatus = onStatus;
    this.mode = mode;
    this.active = false;
    this.complete = false;
    this.attempt = 0;
    this.socket = null;
    this.retry = null;
    this.request = null;
    this.onOnline = () => { this.attempt = 0; this.refresh(); this.connect(); };
    this.onOffline = () => { this.pause(); this.onStatus('offline'); };
    this.onVisibility = () => {
      if (document.hidden) { this.pause(); this.onStatus('paused'); }
      else { this.refresh(); this.connect(); }
    };
  }

  start() {
    if (this.active) return;
    this.active = true;
    window.addEventListener('online', this.onOnline);
    window.addEventListener('offline', this.onOffline);
    document.addEventListener('visibilitychange', this.onVisibility);
    this.connect();
  }

  stop() {
    this.active = false;
    this.pause();
    window.removeEventListener('online', this.onOnline);
    window.removeEventListener('offline', this.onOffline);
    document.removeEventListener('visibilitychange', this.onVisibility);
  }

  abortRequest() {
    if (!this.request) return;
    const request = this.request;
    this.request = null;
    clearTimeout(request.timer);
    request.controller.abort();
  }

  pause() {
    clearTimeout(this.retry);
    this.retry = null;
    const socket = this.socket;
    this.socket = null;
    if (socket) { try { socket.close(1000, 'reader paused'); } catch { /* Already closed. */ } }
    this.abortRequest();
  }

  available() { return this.active && !document.hidden && navigator.onLine; }

  receive(value) {
    const snapshot = parseSnapshot(value);
    this.mode = snapshot.autoUpdatePush;
    this.complete = true;
    this.onSnapshot(snapshot);
  }

  connect() {
    if (!this.active || document.hidden) return;
    if (!navigator.onLine) { this.onStatus('offline'); return; }
    if (this.socket || (this.mode === 0 && this.complete)) {
      if (this.mode === 0) this.onStatus('snapshot');
      return;
    }
    clearTimeout(this.retry); this.retry = null;
    this.onStatus(this.attempt ? 'reconnecting' : 'connecting');
    const endpoint = new URL('/ws?v=1', window.location.href);
    endpoint.protocol = endpoint.protocol === 'https:' ? 'wss:' : 'ws:';
    let socket;
    try { socket = new WebSocket(endpoint); } catch { this.schedule(); return; }
    this.socket = socket;
    socket.onmessage = event => {
      if (this.socket !== socket) return;
      try {
        // A fresh stream message wins over an older in-flight HTTP response.
        const parsed = JSON.parse(event.data);
        this.receive(parsed);
        this.abortRequest();
        this.attempt = 0;
        this.onStatus(this.mode === 0 ? 'snapshot' : 'connected');
      } catch { this.onStatus('invalid'); }
    };
    socket.onclose = event => {
      if (this.socket !== socket) return;
      this.socket = null;
      if (!this.active || document.hidden) return;
      if (this.mode === 0 && this.complete && event.code === 1000) { this.onStatus('snapshot'); return; }
      this.schedule();
    };
    // onclose is the single reconnect path; onerror must not schedule a second timer.
    socket.onerror = () => {};
  }

  schedule() {
    if (!this.available()) { if (this.active && !navigator.onLine) this.onStatus('offline'); return; }
    if (this.mode === 0 && this.complete) { this.onStatus('snapshot'); return; }
    clearTimeout(this.retry);
    this.onStatus('reconnecting');
    const delay = Math.min(30000, 1000 * 2 ** Math.min(this.attempt++, 5)) + Math.floor(Math.random() * 250);
    this.retry = setTimeout(() => { this.retry = null; this.connect(); }, delay);
  }

  async refresh() {
    if (!this.available()) { if (!navigator.onLine) this.onStatus('offline'); return; }
    this.abortRequest();
    const controller = new AbortController();
    const request = { controller, timer: setTimeout(() => controller.abort(), 10000) };
    this.request = request;
    try {
      const response = await fetch('/api/v1/snapshot', { signal: controller.signal, cache: 'no-store', credentials: 'same-origin' });
      if (!response.ok) throw new Error('Snapshot unavailable');
      const value = await response.json();
      if (this.request !== request || !this.active) return;
      this.receive(value);
      this.onStatus(this.mode === 0 ? 'snapshot' : (this.socket?.readyState === WebSocket.OPEN ? 'connected' : 'connecting'));
      if (!this.socket) this.connect();
    } catch {
      if (this.request === request && this.active) this.onStatus(navigator.onLine ? 'unavailable' : 'offline');
    } finally {
      clearTimeout(request.timer);
      if (this.request === request) this.request = null;
    }
  }
}

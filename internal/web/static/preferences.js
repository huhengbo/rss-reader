/* First-paint theme bootstrap. No dependencies, network calls or stored feed data. */
(() => {
  const skins = ['slate', 'paper', 'grove', 'terminal'];
  const modes = ['system', 'light', 'dark'];
  const key = 'rss-reader.appearance.v1';
  function normalize(value) {
    const v = value && typeof value === 'object' ? value : {};
    return { skin: skins.includes(v.skin) ? v.skin : 'slate', mode: modes.includes(v.mode) ? v.mode : 'system', density: v.density === 'compact' ? 'compact' : 'comfortable' };
  }
  function read(storage) {
    try { return normalize(JSON.parse(storage.getItem(key))); } catch { return normalize(null); }
  }
  function save(storage, value) {
    try { storage.setItem(key, JSON.stringify(normalize(value))); } catch { /* UI remains usable in private/blocked storage. */ }
  }
  function apply(value, systemDark = false) {
    const prefs = normalize(value);
    const root = document.documentElement;
    root.dataset.skin = prefs.skin;
    root.dataset.color = prefs.mode === 'system' ? (systemDark ? 'dark' : 'light') : prefs.mode;
    root.dataset.density = prefs.density;
  }
  globalThis.ReaderPreferences = { skins, modes, normalize, read, save, apply };
  if (typeof document !== 'undefined') {
    document.documentElement.classList.add('js');
    let storage;
    try { storage = window.localStorage; } catch { /* Browser may block access itself. */ }
    apply(read(storage), window.matchMedia('(prefers-color-scheme: dark)').matches);
  }
})();

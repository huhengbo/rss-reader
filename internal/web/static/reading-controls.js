// Restore keyboard focus after keyed DOM moves, removals, or visibility changes.
// A connected element can still be inside a hidden article/list item.
export function restoreReadingFocus(active, fallback) {
  if (!active || active === document.body || active === document.documentElement) return;
  const visible = element => element?.isConnected && !element.disabled &&
    !element.closest('[hidden]') && element.getClientRects().length > 0 &&
    getComputedStyle(element).visibility !== 'hidden';
  const target = [active, fallback, document.getElementById('main')].find(visible);
  if (target && document.activeElement !== target) target.focus({ preventScroll: true });
}

// The same native controls serve desktop and mobile. Only their disclosure
// changes; resizing must not rebuild the DOM, reset preferences, or reconnect.
export function initializeDisplaySettings() {
  const settings = document.getElementById('display-settings');
  const summary = settings.querySelector('summary');
  const mobile = matchMedia('(max-width: 760px), (max-width: 900px) and (max-height: 500px)');
  let mobileOpen = false;
  const synchronize = () => {
    const active = document.activeElement;
    settings.open = !mobile.matches || mobileOpen;
    if (!settings.open && settings.contains(active) && active !== summary) {
      summary.focus({ preventScroll: true });
    } else if (!mobile.matches && active === summary) {
      document.getElementById('search').focus({ preventScroll: true });
    }
  };
  settings.addEventListener('toggle', () => {
    if (mobile.matches) mobileOpen = settings.open;
  });
  mobile.addEventListener('change', synchronize);
  synchronize();
}

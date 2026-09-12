import { parseSnapshot, contentKey, newItemCount } from './model.js';
import { ReaderConnection } from './connection.js';

const $ = selector => document.querySelector(selector);
const text = (element, value) => { if (element.textContent !== value) element.textContent = value; };
const statusLabels = { loading: '加载中', ready: '已同步', empty: '暂无文章', error: '获取失败', stale: '旧缓存' };

// Reuse keyed nodes; filter/theme/status updates must not replace the whole page.
function reconcile(parent, nodes) {
  const wanted = new Set(nodes);
  for (const node of [...parent.children]) if (!wanted.has(node)) node.remove();
  let cursor = parent.firstElementChild;
  for (const node of nodes) {
    if (node === cursor) cursor = cursor.nextElementSibling;
    else parent.insertBefore(node, cursor);
  }
}

function updateTime(element, stamp, empty = '—') {
  const date = new Date(stamp);
  if (!stamp || !Number.isFinite(date.getTime())) { text(element, empty); element.removeAttribute('datetime'); element.removeAttribute('title'); return; }
  element.dateTime = stamp;
  element.title = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', timeZoneName: 'short' }).format(date);
  const minutes = Math.round((date.getTime() - Date.now()) / 60000);
  text(element, Math.abs(minutes) < 1 ? '刚刚' : new Intl.RelativeTimeFormat('zh-CN', { numeric: 'auto' }).format(Math.abs(minutes) < 60 ? minutes : Math.round(minutes / 60), Math.abs(minutes) < 60 ? 'minute' : 'hour'));
}

function boot() {
  let shown = parseSnapshot(JSON.parse($('#initial-data').textContent));
  let pending = null;
  let query = '', selected = '';
  const expanded = new Set();
  let storage;
  try { storage = window.localStorage; } catch { /* Storage is optional. */ }
  let collapsed = new Set();
  try {
    const saved = JSON.parse(storage.getItem('rss-reader.collapsed.v1'));
    if (Array.isArray(saved)) collapsed = new Set(saved.filter(id => typeof id === 'string' && /^[a-zA-Z0-9_-]{1,128}$/.test(id)).slice(0, 1000));
  } catch { /* Invalid preferences do not affect reading. */ }
  const saveCollapsed = () => { try { storage.setItem('rss-reader.collapsed.v1', JSON.stringify([...collapsed].slice(0, 1000))); } catch { /* Optional. */ } };
  const appearance = globalThis.ReaderPreferences;
  let prefs = appearance.read(storage);
  const media = window.matchMedia('(prefers-color-scheme: dark)');
  for (const key of ['skin', 'mode', 'density']) {
    $(`#${key}`).value = prefs[key];
    $(`#${key}`).addEventListener('change', event => {
      prefs = appearance.normalize({ ...prefs, [key]: event.target.value });
      appearance.apply(prefs, media.matches);
      appearance.save(storage, prefs);
    });
  }
  const followSystem = () => appearance.apply(prefs, media.matches);
  media.addEventListener('change', followSystem);

  const sourceTemplate = $('#source-template');
  const itemTemplate = $('#item-template');
  const grid = $('#sources');
  const cards = new Map([...grid.children].map(card => [card.dataset.id, card]));
  const navButtons = new Map();
  const options = new Map();

  function render() {
    const active = document.activeElement;
    const anchor = [...grid.children].find(card => !card.hidden && card.getBoundingClientRect().bottom > 0);
    const anchorTop = anchor?.getBoundingClientRect().top;
    const wantedIds = new Set(shown.sources.map(source => source.id));
    if (selected && !wantedIds.has(selected)) selected = '';
    for (const id of collapsed) if (!wantedIds.has(id)) collapsed.delete(id);
    for (const id of expanded) if (!wantedIds.has(id)) expanded.delete(id);
    text($('#page-title'), shown.title); document.title = shown.title;
    text($('#page-description'), shown.description);
    text($('#source-count'), String(shown.sources.length));
    $('#main').style.setProperty('--list-height', `${shown.listHeight}px`);
    const sources = [{ id: '', title: '全部订阅源' }, ...shown.sources];
    const buttons = [], selects = [];
    for (const source of sources) {
      let button = navButtons.get(source.id);
      if (!button) { button = document.createElement('button'); button.type = 'button'; button.dataset.source = source.id; button.className = 'source-button'; navButtons.set(source.id, button); }
      text(button, source.title); button.setAttribute('aria-pressed', String(selected === source.id)); buttons.push(button);
      let option = options.get(source.id);
      if (!option) { option = document.createElement('option'); option.value = source.id; options.set(source.id, option); }
      text(option, source.title); selects.push(option);
    }
    reconcile($('#source-nav'), buttons); reconcile($('#source-filter'), selects); $('#source-filter').value = selected;
    for (const id of navButtons.keys()) if (id && !wantedIds.has(id)) { navButtons.delete(id); options.delete(id); }

    let resultSources = 0, resultItems = 0;
    const nodes = [];
    const search = query.trim().toLocaleLowerCase();
    for (const source of shown.sources) {
      let card = cards.get(source.id);
      if (!card) { card = sourceTemplate.content.firstElementChild.cloneNode(true); card.dataset.id = source.id; cards.set(source.id, card); }
      card.setAttribute('aria-labelledby', `heading-${source.id}`);
      const heading = card.querySelector('[data-part="title"]'); heading.id = `heading-${source.id}`; text(heading, source.title);
      text(card.querySelector('.feed-avatar'), [...source.title].slice(0, 1).join(''));
      const state = card.querySelector('.source-state'); state.dataset.status = source.status; text(state, statusLabels[source.status]);
      const message = card.querySelector('.source-message'); message.hidden = !source.message; text(message, source.message);
      card.querySelector('.skeleton').hidden = source.status !== 'loading';
      const isCollapsed = collapsed.has(source.id);
      const body = card.querySelector('.card-body'); body.id = `body-${source.id}`; body.hidden = isCollapsed;
      const toggle = card.querySelector('[data-action="collapse"]');
      toggle.setAttribute('aria-controls', body.id); toggle.setAttribute('aria-expanded', String(!isCollapsed));
      toggle.setAttribute('aria-label', `${isCollapsed ? '展开' : '折叠'} ${source.title}`); text(toggle, isCollapsed ? '+' : '−');
      const sourceMatches = source.title.toLocaleLowerCase().includes(search);
      const matches = source.items.filter(item => !search || sourceMatches || item.title.toLocaleLowerCase().includes(search));
      const visible = (!selected || source.id === selected) && (!search || sourceMatches || matches.length > 0);
      card.hidden = !visible;
      if (visible) { resultSources++; resultItems += matches.length; }
      const showAll = expanded.has(source.id) || Boolean(search);
      card.classList.toggle('expanded', showAll);
      const matchIDs = new Set((showAll ? matches : matches.slice(0, 8)).map(item => item.id));
      const list = card.querySelector('.article-list');
      const oldItems = new Map([...list.children].map(li => [li.dataset.itemId, li]));
      const items = [];
      for (const item of source.items) {
        let li = oldItems.get(item.id);
        if (!li) { li = itemTemplate.content.firstElementChild.cloneNode(true); li.dataset.itemId = item.id; }
        li.hidden = !matchIDs.has(item.id);
        const link = li.querySelector('a');
        if (item.link) link.setAttribute('href', item.link); else link.removeAttribute('href');
        text(li.querySelector('.article-title'), item.title);
        const time = li.querySelector('time'); time.hidden = !item.publishedAt; updateTime(time, item.publishedAt);
        items.push(li);
      }
      reconcile(list, items);
      const more = card.querySelector('[data-action="more"]'); more.hidden = matches.length <= 8 || Boolean(search);
      text(more, showAll ? '收起列表' : `展开更多（共 ${matches.length} 篇）`);
      more.setAttribute('aria-expanded', String(showAll));
      const empty = card.querySelector('.empty-source'); empty.hidden = source.items.length > 0;
      text(empty, source.status === 'loading' ? '首次同步中，内容就绪后会显示在这里。' : source.status === 'error' ? '暂时无法取得内容，请检查订阅源配置。' : '这个订阅源暂时没有文章。');
      updateTime(card.querySelector('[data-part="success"]'), source.lastSuccessAt, '尚未成功');
      updateTime(card.querySelector('[data-part="changed"]'), source.contentChangedAt);
      nodes.push(card);
    }
    reconcile(grid, nodes);
    for (const id of cards.keys()) if (!wantedIds.has(id)) cards.delete(id);
    text($('#result-count'), `${resultSources} 个订阅源 · ${resultItems} 篇文章`);
    $('#empty-state').hidden = resultSources > 0;
    text($('#empty-title'), shown.sources.length ? '没有匹配的内容' : '还没有订阅源');
    text($('#empty-description'), shown.sources.length ? '换个关键词，或清空搜索并选择全部订阅源。' : '请在配置文件的 values 中添加订阅源。');
    if (anchor?.isConnected && !anchor.hidden && window.scrollY > 0) window.scrollBy(0, anchor.getBoundingClientRect().top - anchorTop);
    if (active && !active.isConnected) $('#main').focus({ preventScroll: true });
  }

  function receive(snapshot) {
    if (snapshot.revision === shown.revision && !pending) return;
    const structureChanged = shown.sources.map(s => s.id).join(',') !== snapshot.sources.map(s => s.id).join(',');
    const initialLoading = shown.sources.some(s => s.status === 'loading');
    if (structureChanged || initialLoading || contentKey(shown) === contentKey(snapshot)) {
      pending = null; shown = snapshot; $('#updates').hidden = true; render(); return;
    }
    pending = snapshot;
    const count = newItemCount(shown, snapshot);
    text($('#update-text'), count ? `有 ${count} 篇新文章，准备好再应用。` : '订阅内容有变化，应用后更新当前列表。');
    $('#updates').hidden = false;
    // Status feedback is immediate, but the currently read articles stay in place.
    const fresh = new Map(snapshot.sources.map(s => [s.id, s]));
    shown = { ...snapshot, sources: shown.sources.map(old => ({ ...fresh.get(old.id), title: old.title, link: old.link, items: old.items, contentChangedAt: old.contentChangedAt })) };
    render();
  }

  const connection = new ReaderConnection({ mode: shown.autoUpdatePush, onSnapshot: receive, onStatus: status => {
    const labels = { connecting: '正在连接…', connected: '实时连接正常', reconnecting: '连接中断，正在重连；已显示内容仍可阅读', offline: '当前离线，联网后自动恢复', paused: '页面已暂停实时连接', snapshot: '快照模式 · 点击刷新视图读取最新缓存', invalid: '收到无效数据，已忽略；现有内容保留', unavailable: '暂时无法获取最新视图；现有内容保留' };
    $('#connection').dataset.state = status;
    text($('#connection'), labels[status] || status);
  } });
  $('#search').addEventListener('input', event => { query = event.target.value; render(); });
  $('#clear-search').addEventListener('click', () => { query = ''; selected = ''; $('#search').value = ''; render(); $('#search').focus(); });
  $('#source-filter').addEventListener('change', event => { selected = event.target.value; render(); });
  $('#source-nav').addEventListener('click', event => { const button = event.target.closest('button[data-source]'); if (button) { selected = button.dataset.source; render(); } });
  grid.addEventListener('click', event => {
    const button = event.target.closest('button[data-action]'); if (!button) return;
    const id = button.closest('[data-id]').dataset.id;
    const set = button.dataset.action === 'collapse' ? collapsed : expanded;
    if (set.has(id)) set.delete(id); else set.add(id);
    render(); if (set === collapsed) saveCollapsed();
  });
  $('#apply-updates').addEventListener('click', () => { if (!pending) return; shown = pending; pending = null; $('#updates').hidden = true; render(); $('#refresh-view').focus({ preventScroll: true }); });
  $('#refresh-view').addEventListener('click', async () => {
    const button = $('#refresh-view'); button.disabled = true;
    try { await connection.refresh(); } finally { button.disabled = false; }
  });
  let clock;
  const start = () => {
    connection.start(); clearInterval(clock);
    clock = setInterval(() => { for (const time of grid.querySelectorAll('time[datetime]')) updateTime(time, time.dateTime); }, 60000);
  };
  window.addEventListener('pagehide', () => { connection.stop(); clearInterval(clock); media.removeEventListener('change', followSystem); });
  window.addEventListener('pageshow', event => { if (event.persisted) { media.addEventListener('change', followSystem); followSystem(); start(); connection.refresh(); } });
  render(); document.documentElement.classList.add('hydrated'); start();
}

try { boot(); } catch {
  document.documentElement.classList.remove('js');
  const message = $('#page-error');
  message.hidden = false; text(message, '交互界面暂时未能初始化，仍可阅读下方静态内容。');
}

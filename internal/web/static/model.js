export function safeLink(value) {
  if (typeof value !== 'string') return '';
  try {
    const url = new URL(value);
    return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.href : '';
  } catch { return ''; }
}

export function parseSnapshot(value) {
  if (!value || value.version !== 1 || typeof value.revision !== 'string' || !Array.isArray(value.sources) || !Number.isInteger(value.autoUpdatePush) || value.autoUpdatePush < 0) throw new Error('Invalid snapshot');
  const ids = new Set();
  const statuses = ['loading', 'ready', 'empty', 'error', 'stale'];
  const text = v => typeof v === 'string' ? v : '';
  const identity = v => typeof v === 'string' && /^[a-zA-Z0-9_-]{1,128}$/.test(v);
  const sources = value.sources.map(source => {
    if (!source || !identity(source.id) || ids.has(source.id) || !statuses.includes(source.status) || !Array.isArray(source.items)) throw new Error('Invalid source');
    ids.add(source.id);
    const itemIds = new Set();
    const items = source.items.map(item => {
      if (!item || !identity(item.id) || itemIds.has(item.id)) throw new Error('Invalid item');
      itemIds.add(item.id);
      return { id: item.id, title: text(item.title) || '无标题文章', link: safeLink(item.link), publishedAt: text(item.publishedAt) };
    });
    return { id: source.id, title: text(source.title) || '订阅源', link: safeLink(source.link), status: source.status, message: text(source.message), lastAttemptAt: text(source.lastAttemptAt), lastSuccessAt: text(source.lastSuccessAt), contentChangedAt: text(source.contentChangedAt), items };
  });
  return { version: 1, revision: value.revision, title: text(value.title) || 'RSS Reader', description: text(value.description), autoUpdatePush: value.autoUpdatePush, listHeight: Math.max(180, Math.min(1600, Number(value.listHeight) || 600)), sources };
}

export function contentKey(snapshot) {
  return JSON.stringify(snapshot.sources.map(source => [source.id, source.title, source.link, source.items.map(item => [item.id, item.title, item.link, item.publishedAt || ''])]));
}

export function newItemCount(older, newer) {
  const known = new Set(older.sources.flatMap(source => source.items.map(item => `${source.id}:${item.id}`)));
  return newer.sources.reduce((total, source) => total + source.items.filter(item => !known.has(`${source.id}:${item.id}`)).length, 0);
}

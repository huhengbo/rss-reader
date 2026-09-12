import { test, expect } from '@playwright/test';

async function openReader(page) {
  await page.goto('/');
  await expect(page.locator('html')).toHaveClass(/hydrated/);
}

test.beforeEach(async ({ page, request }) => {
  await request.post('/fixture/reset');
  await page.clock.setFixedTime(new Date('2026-09-12T00:10:00Z'));
  page.uiErrors = [];
  page.externalRequests = [];
  page.on('pageerror', error => page.uiErrors.push(error.message));
  await page.route('**/*', route => {
    if (new URL(route.request().url()).hostname !== '127.0.0.1') { page.externalRequests.push(route.request().url()); return route.abort(); }
    return route.continue();
  });
});

test.afterEach(async ({ page }) => {
  expect(page.uiErrors).toEqual([]);
  expect(page.externalRequests).toEqual([]);
});

test('same-homepage sources survive SSR handover; search, filtering and keyboard controls work', async ({ page }) => {
  await openReader(page);
  await expect(page.locator('#sources > article')).toHaveCount(3);
  await expect(page.getByRole('heading', { name: '技术周刊', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: '同站专题', exact: true })).toBeVisible();
  await page.getByRole('searchbox').fill('不存在的文章');
  await expect(page.getByRole('heading', { name: '没有匹配的内容' })).toBeVisible();
  await page.getByRole('button', { name: '清空', exact: true }).click();
  await expect(page.getByRole('searchbox')).toBeFocused();
  await page.getByRole('combobox', { name: '筛选订阅源' }).selectOption({ label: '同站专题' });
  await expect(page.locator('#sources > article:visible')).toHaveCount(1);
  const toggle = page.getByRole('button', { name: '折叠 同站专题', exact: true });
  await toggle.focus(); await page.keyboard.press('Space');
  await expect(page.getByRole('button', { name: '展开 同站专题', exact: true })).toHaveAttribute('aria-expanded', 'false');
  await page.keyboard.press('Space');
  await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  await expect(toggle).toBeFocused();
});

test('new articles require confirmation; source removal and empty config are authoritative', async ({ page, request }) => {
  await openReader(page);
  await request.post('/fixture/new');
  await expect(page.locator('#updates')).toBeVisible();
  await expect(page.getByText('新文章：可靠的实时阅读', { exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '应用更新' }).click();
  const card = page.locator('#sources > article').filter({ has: page.getByRole('heading', { name: '技术周刊', exact: true }) });
  await card.getByRole('button', { name: /展开更多/ }).click();
  await expect(page.getByRole('link', { name: '新文章：可靠的实时阅读' })).toBeVisible();
  await request.post('/fixture/delete');
  await expect(page.locator('#sources > article')).toHaveCount(2);
  await expect(page.getByRole('heading', { name: '技术周刊', exact: true })).toHaveCount(0);
  await request.post('/fixture/empty');
  await expect(page.getByRole('heading', { name: '还没有订阅源' })).toBeVisible();
  await expect(page.locator('#sources > article')).toHaveCount(0);
});

test('source failure preserves articles with honest stale status', async ({ page, request }) => {
  await openReader(page);
  await request.post('/fixture/fail');
  await expect(page.getByText('同步失败，正在显示上次成功的内容')).toBeVisible();
  await expect(page.getByRole('link', { name: /技术周刊：第 1 篇/ })).toBeVisible();
  await expect(page.locator('body')).not.toContainText('do not disclose');
});

test('native offline/online recovery preserves document, search and appearance on mobile', async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let navigations = 0;
  page.on('framenavigated', frame => { if (frame === page.mainFrame()) navigations++; });
  await openReader(page);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  await page.getByRole('searchbox').fill('技术');
  await page.locator('#skin').selectOption('paper');
  await context.setOffline(true);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'offline');
  await context.setOffline(false);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
  await expect(page.locator('html')).toHaveAttribute('data-skin', 'paper');
  expect(navigations).toBe(1);
});

test('appearance is persistent, follows system only when selected and does not recreate sockets', async ({ page }) => {
  let sockets = 0; page.on('websocket', () => sockets++);
  await openReader(page);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  const before = sockets;
  await page.locator('#skin').selectOption('grove');
  await page.locator('#density').selectOption('compact');
  await page.locator('#mode').selectOption('dark');
  await page.emulateMedia({ colorScheme: 'light' });
  await expect(page.locator('html')).toHaveAttribute('data-color', 'dark');
  expect(sockets).toBe(before);
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-skin', 'grove');
  await expect(page.locator('html')).toHaveAttribute('data-density', 'compact');
  await page.locator('#mode').selectOption('system');
  await expect(page.locator('html')).toHaveAttribute('data-color', 'light');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveAttribute('data-color', 'dark');
});

test('blocked browser storage does not prevent rendering or changing skin', async ({ page }) => {
  await page.addInitScript(() => Object.defineProperty(window, 'localStorage', { get() { throw new DOMException('blocked', 'SecurityError'); } }));
  await openReader(page);
  await page.locator('#skin').selectOption('terminal');
  await expect(page.locator('html')).toHaveAttribute('data-skin', 'terminal');
  await expect(page.locator('#page-error')).toBeHidden();
});

test('invalid stream frames are isolated; valid messages still work', async ({ page, request }) => {
  const snapshot = await (await request.get('/api/v1/snapshot')).json();
  let stream;
  await page.routeWebSocket('**/ws?v=1', ws => { stream = ws; ws.send(JSON.stringify(snapshot)); });
  await openReader(page);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  stream.send('not JSON');
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'invalid');
  stream.send(JSON.stringify(snapshot));
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  await expect(page.locator('#sources > article')).toHaveCount(3);
});

test('one-shot snapshot mode does not enter a reconnect loop', async ({ page, request }) => {
  await request.post('/fixture/zero');
  let sockets = 0; page.on('websocket', () => sockets++);
  await page.clock.install();
  await openReader(page);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'snapshot');
  await page.clock.fastForward(65000);
  expect(sockets).toBe(1);
});

test('four skins and two modes remain readable at narrow and desktop widths', async ({ page }, testInfo) => {
  test.setTimeout(120000);
  await openReader(page);
  for (const skin of ['slate', 'paper', 'grove', 'terminal']) {
    await page.locator('#skin').selectOption(skin);
    for (const mode of ['light', 'dark']) {
      await page.locator('#mode').selectOption(mode);
      const ratios = await page.evaluate(() => {
        const css = getComputedStyle(document.documentElement);
        const luminance = token => {
          const hex = css.getPropertyValue(token).trim().slice(1);
          const rgb = [0, 2, 4].map(i => parseInt(hex.slice(i, i+2), 16) / 255).map(v => v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4);
          return rgb[0] * .2126 + rgb[1] * .7152 + rgb[2] * .0722;
        };
        return [['--text','--surface'], ['--muted','--surface'], ['--muted','--subtle'], ['--accent','--surface'], ['--on-accent','--accent'], ['--success','--success-soft'], ['--warning','--warning-soft'], ['--danger','--danger-soft']].map(([a,b]) => { const x=luminance(a),y=luminance(b); return (Math.max(x,y)+.05)/(Math.min(x,y)+.05); });
      });
      for (const ratio of ratios) expect(ratio, `${skin}/${mode} semantic text contrast`).toBeGreaterThanOrEqual(4.5);
      for (const width of [320, 390, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${skin}/${mode}/${width}`).toBe(true);
        if (width === 390) expect(await page.locator('.article-list').first().evaluate(el => getComputedStyle(el).overflowY)).toBe('visible');
        if (width !== 320) await testInfo.attach(`${skin}-${mode}-${width}`, { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' });
      }
    }
  }
});

test('long text, reduced motion and skip navigation remain usable', async ({ page, request }) => {
  await request.post('/fixture/long');
  await page.setViewportSize({ width: 320, height: 844 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await openReader(page);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  expect(await page.locator('#refresh-view').evaluate(el => getComputedStyle(el).transitionDuration)).toBe('0s');
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: '跳到订阅内容' })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('#main')).toBeFocused();
});

test('server-rendered content is usable without JavaScript', async ({ browser, request }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  try {
    const page = await context.newPage();
    await page.goto('http://127.0.0.1:4173/');
    await expect(page.getByRole('heading', { name: '技术周刊', exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: /技术周刊：第 12 篇/ })).toBeVisible();
    await expect(page.getByText(/当前为静态阅读模式/)).toBeVisible();
    expect((await (await request.get('/api/v1/snapshot')).text()).includes('fixture-private')).toBe(false);
  } finally { await context.close(); }
});

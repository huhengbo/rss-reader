import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page, request }) => {
  await request.post('/fixture/reset');
  page.uiErrors = [];
  page.on('pageerror', error => page.uiErrors.push(error.message));
});
test.afterEach(async ({ page }) => { expect(page.uiErrors).toEqual([]); });

async function openReader(page) {
  await page.goto('/');
  await expect(page.locator('html')).toHaveClass(/hydrated/);
}

test('mobile disclosure supports keyboard operation and keeps one set of controls', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 568 });
  await openReader(page);
  const settings = page.locator('#display-settings');
  const summary = settings.locator('summary');
  await expect(settings).not.toHaveAttribute('open');
  await expect(page.locator('#skin')).toBeHidden();
  await page.getByRole('searchbox').fill('技术');
  await page.getByRole('searchbox').press('Enter');
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
  await summary.focus();
  await page.keyboard.press('Enter');
  await expect(settings).toHaveAttribute('open');
  for (const id of ['skin', 'mode', 'density', 'source-filter']) {
    await page.keyboard.press('Tab');
    await expect(page.locator(`#${id}`)).toBeFocused();
    await expect(page.locator(`#${id}`)).toHaveCount(1);
    await expect(page.locator(`#${id}`)).toHaveCSS('font-size', '16px');
  }
  await summary.focus();
  await page.keyboard.press('Space');
  await expect(settings).not.toHaveAttribute('open');
  await expect(summary).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('button', { name: '折叠 技术周刊', exact: true })).toBeFocused();
  await page.setViewportSize({ width: 1280, height: 900 });
  await expect(settings).toHaveAttribute('open');
  await expect(summary).toBeHidden();
  await expect(page.locator('#skin')).toBeVisible();
  await page.locator('#skin').focus();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(settings).not.toHaveAttribute('open');
  await expect(summary).toBeFocused();
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
});

for (const hideFocusedItem of [false, true]) {
  test(`source reorder restores ${hideFocusedItem ? 'visible fallback for a now-hidden item' : 'the same article focus'}`, async ({ page, request }) => {
    const snapshot = await (await request.get('/api/v1/snapshot')).json();
    let stream;
    await page.routeWebSocket('**/ws?v=1', ws => { stream = ws; ws.send(JSON.stringify(snapshot)); });
    await openReader(page);
    await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
    const source = snapshot.sources.find(source => source.items.length > 8);
    expect(source).toBeTruthy();
    const item = source.items[7];
    const card = page.locator(`[data-id="${source.id}"]`);
    const link = card.locator(`[data-item-id="${item.id}"] a`);
    await link.focus();
    await expect(link).toBeFocused();
    const next = structuredClone(snapshot);
    next.revision += '-reordered';
    next.sources.reverse();
    if (hideFocusedItem) {
      next.sources.find(s => s.id === source.id).items.unshift({ ...item, id: 'focus-regression-new', title: '新插入的文章' });
    }
    stream.send(JSON.stringify(next));
    await expect(page.locator('#sources > article').first()).toHaveAttribute('data-id', next.sources[0].id);
    if (hideFocusedItem) {
      await expect(link).toBeHidden();
      expect(await link.evaluate(element => element.isConnected)).toBe(true);
      await expect(card.locator('[data-action="collapse"]')).toBeFocused();
    } else {
      await expect(link).toBeFocused();
    }
  });
}

test('refresh retains keyboard focus, prevents duplicate requests and does not steal focus on completion', async ({ page, request }) => {
  const snapshot = await (await request.get('/api/v1/snapshot')).json();
  await openReader(page);
  let finish, requests = 0;
  const pending = new Promise(resolve => { finish = resolve; });
  await page.route('**/api/v1/snapshot', async route => {
    requests++;
    await pending;
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(snapshot) });
  });
  const refresh = page.locator('#refresh-view');
  try {
    await refresh.focus();
    await page.keyboard.press('Enter');
    await expect(refresh).toHaveAttribute('aria-busy', 'true');
    await expect(refresh).toHaveAttribute('aria-disabled', 'true');
    await expect(refresh).toBeFocused();
    await page.keyboard.press('Enter');
    await expect.poll(() => requests).toBe(1);
    await page.keyboard.press('Tab');
    await expect(page.locator('#skin')).toBeFocused();
  } finally { finish(); }
  await expect(refresh).not.toHaveAttribute('aria-busy');
  await expect(refresh).not.toHaveAttribute('aria-disabled');
  await expect(page.locator('#skin')).toBeFocused();
  expect(requests).toBe(1);
});

test('expand and collapse keep focus and expose the controlled native list', async ({ page }) => {
  await openReader(page);
  const card = page.getByRole('article', { name: '技术周刊', exact: true });
  const more = card.locator('[data-action="more"]');
  const list = card.getByRole('list');
  await expect(more).toHaveAttribute('aria-controls', await list.getAttribute('id'));
  await more.focus();
  await page.keyboard.press('Enter');
  await expect(more).toHaveAttribute('aria-expanded', 'true');
  await expect(list.getByRole('listitem')).toHaveCount(12);
  await expect(more).toBeFocused();
  await page.keyboard.press('Space');
  await expect(more).toHaveAttribute('aria-expanded', 'false');
  await expect(list.getByRole('listitem')).toHaveCount(8);
  await expect(more).toBeFocused();
});

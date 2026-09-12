import { test, expect } from '@playwright/test';

const portraitViewports = [
  { width: 320, height: 568 },
  { width: 360, height: 800 },
  { width: 375, height: 812 },
  { width: 390, height: 844 },
  { width: 430, height: 932 }
];

async function openReader(page, viewport) {
  await page.setViewportSize(viewport);
  await page.goto('/');
  await expect(page.locator('html')).toHaveClass(/hydrated/);
}

async function expectNoHorizontalOverflow(page) {
  const metrics = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    innerWidth: window.innerWidth
  }));
  expect(metrics.scrollWidth, `scrollWidth ${metrics.scrollWidth} > innerWidth ${metrics.innerWidth}`).toBeLessThanOrEqual(metrics.innerWidth + 1);
}

test.beforeEach(async ({ page, request }) => {
  await request.post('/fixture/reset');
  page.uiErrors = [];
  page.externalRequests = [];
  page.on('pageerror', error => page.uiErrors.push(error.message));
  await page.route('**/*', route => {
    if (new URL(route.request().url()).hostname !== '127.0.0.1') {
      page.externalRequests.push(route.request().url());
      return route.abort();
    }
    return route.continue();
  });
});

test.afterEach(async ({ page }) => {
  expect(page.uiErrors).toEqual([]);
  expect(page.externalRequests).toEqual([]);
});

for (const viewport of portraitViewports) {
  test(`portrait ${viewport.width}x${viewport.height} keeps a single readable scroll surface`, async ({ page, request }) => {
    await request.post('/fixture/long');
    await openReader(page, viewport);

    await expectNoHorizontalOverflow(page);
    await expect(page.locator('.sidebar nav')).toBeHidden();
    await expect(page.getByRole('searchbox')).toBeVisible();
    await expect(page.locator('.article-list').first()).toHaveCSS('overflow-y', 'visible');

    const viewportMeta = await page.locator('meta[name="viewport"]').getAttribute('content');
    expect(viewportMeta).toContain('viewport-fit=cover');

    const undersizedTargets = await page.locator('button:visible, input:visible, select:visible').evaluateAll(elements =>
      elements
        .map(element => ({ label: element.getAttribute('aria-label') || element.id || element.textContent?.trim(), height: element.getBoundingClientRect().height }))
        .filter(target => target.height < 43.5)
    );
    expect(undersizedTargets).toEqual([]);
  });
}

test('phone landscape keeps mobile navigation and preserves reading state across rotation', async ({ page }) => {
  await openReader(page, { width: 390, height: 844 });
  await page.getByRole('searchbox').fill('技术');
  await page.locator('#skin').selectOption('paper');
  await page.locator('#density').selectOption('compact');

  const card = page.locator('#sources > article').filter({ has: page.getByRole('heading', { name: '技术周刊', exact: true }) });
  const toggle = card.locator('button[data-action="collapse"]');
  await expect(toggle).toHaveAccessibleName('折叠 技术周刊');
  await toggle.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(toggle).toHaveAccessibleName('展开 技术周刊');

  await page.setViewportSize({ width: 844, height: 390 });
  await expectNoHorizontalOverflow(page);
  await expect(page.locator('.sidebar nav')).toBeHidden();
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
  await expect(page.locator('html')).toHaveAttribute('data-skin', 'paper');
  await expect(page.locator('html')).toHaveAttribute('data-density', 'compact');
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(toggle).toHaveAccessibleName('展开 技术周刊');

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(toggle).toHaveAccessibleName('展开 技术周刊');
});

test('search remains usable when the visual area becomes keyboard-sized', async ({ page }) => {
  await openReader(page, { width: 390, height: 844 });
  const search = page.getByRole('searchbox');
  await search.focus();
  await search.fill('周刊');

  await page.setViewportSize({ width: 390, height: 480 });
  const box = await search.boundingBox();
  expect(box).not.toBeNull();
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.y + box.height).toBeLessThanOrEqual(480);
  await expect(search).toBeFocused();
  await expect(search).toHaveValue('周刊');

  await page.getByRole('button', { name: '清空', exact: true }).click();
  await expect(search).toBeFocused();
  await expect(search).toHaveValue('');
});

import { test, expect } from '@playwright/test';

async function openReader(page) {
  await page.goto('/');
  await expect(page.locator('html')).toHaveClass(/hydrated/);
}

async function expectNoHorizontalOverflow(page) {
  const { scrollWidth, innerWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    innerWidth: window.innerWidth
  }));
  expect(scrollWidth).toBeLessThanOrEqual(innerWidth + 1);
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

test('mobile profile uses touch input without hover-only interaction', async ({ page }, testInfo) => {
  await openReader(page);

  const capabilities = await page.evaluate(() => ({
    touchPoints: navigator.maxTouchPoints,
    coarse: matchMedia('(pointer: coarse)').matches,
    fineHover: matchMedia('(hover: hover) and (pointer: fine)').matches,
    mobileUA: /Android|iPhone|Mobile/i.test(navigator.userAgent)
  }));

  // Linux WebKit accepts tap() but may still expose maxTouchPoints as 0.
  // Keep Android assertions strict; do not fake properties to make WebKit green.
  expect(testInfo.project.use.hasTouch).toBe(true);
  expect(capabilities.fineHover).toBe(false);
  expect(capabilities.mobileUA).toBe(true);
  if (testInfo.project.name === 'mobile-chromium') {
    expect(capabilities.touchPoints).toBeGreaterThan(0);
    expect(capabilities.coarse).toBe(true);
  }

  await expectNoHorizontalOverflow(page);
  await expect(page.locator('.sidebar nav')).toBeHidden();
  await expect(page.locator('.article-list').first()).toHaveCSS('overflow-y', 'visible');
  await expect(page.getByRole('button', { name: '清空', exact: true })).toHaveCSS('touch-action', 'manipulation');
  await expect(page.locator('#display-settings')).not.toHaveAttribute('open');
  await page.locator('#display-settings > summary').tap();
  await expect(page.locator('#source-filter')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const undersizedTargets = await page.locator('button:visible, input:visible, select:visible, summary:visible').evaluateAll(elements =>
    elements
      .map(element => ({ label: element.getAttribute('aria-label') || element.id || element.textContent?.trim(), height: element.getBoundingClientRect().height }))
      .filter(target => target.height < 43.5)
  );
  expect(undersizedTargets).toEqual([]);
});

test('touch interactions and rotation-sized resize preserve state', async ({ page }) => {
  let sockets = 0;
  page.on('websocket', () => sockets++);
  await openReader(page);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  const initialSockets = sockets;
  const search = page.getByRole('searchbox');
  await search.fill('技术');
  await page.locator('#display-settings > summary').tap();
  await page.locator('#skin').selectOption('grove');
  await page.locator('#density').selectOption('compact');
  await page.locator('#source-filter').selectOption({ label: '技术周刊' });
  await page.locator('#display-settings > summary').tap();
  await expect(page.locator('#skin')).toBeHidden();

  const card = page.locator('#sources > article').filter({ has: page.getByRole('heading', { name: '技术周刊', exact: true }) });
  const toggle = card.locator('button[data-action="collapse"]');
  await toggle.tap();
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(toggle).toHaveAccessibleName('展开 技术周刊');

  const viewport = page.viewportSize();
  expect(viewport).not.toBeNull();
  await page.setViewportSize({ width: viewport.height, height: viewport.width });
  await expectNoHorizontalOverflow(page);
  await expect(page.locator('.sidebar nav')).toBeHidden();
  await expect(search).toHaveValue('技术');
  await expect(page.locator('html')).toHaveAttribute('data-skin', 'grove');
  await expect(page.locator('html')).toHaveAttribute('data-density', 'compact');
  await expect(page.locator('#display-settings')).not.toHaveAttribute('open');
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');

  await page.setViewportSize(viewport);
  await page.locator('#display-settings > summary').tap();
  await expect(page.locator('#source-filter option:checked')).toHaveText('技术周刊');
  await page.locator('#display-settings > summary').tap();
  await page.getByRole('button', { name: '清空', exact: true }).tap();
  await expect(search).toHaveValue('');
  await expect(search).toBeFocused();
  expect(sockets).toBe(initialSockets);
});

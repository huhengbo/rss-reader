import { test, expect, chromium } from '@playwright/test';
import { fileURLToPath } from 'node:url';

const origin = 'http://127.0.0.1:4173';

async function setZoom(worker, factor) {
  return worker.evaluate(async ({ origin, factor }) => {
    const tabs = (await chrome.tabs.query({})).filter(tab => tab.url && new URL(tab.url).origin === origin);
    if (tabs.length !== 1) throw new Error('Expected exactly one local reader tab');
    await chrome.tabs.setZoomSettings(tabs[0].id, { mode: 'automatic', scope: 'per-tab' });
    await chrome.tabs.setZoom(tabs[0].id, factor);
    return chrome.tabs.getZoom(tabs[0].id);
  }, { origin, factor });
}

async function metrics(page) {
  return page.evaluate(() => ({
    innerWidth, outerWidth, pixelRatio: devicePixelRatio,
    scrollWidth: document.documentElement.scrollWidth,
    pinchScale: visualViewport.scale,
    cssZoom: getComputedStyle(document.documentElement).zoom,
    bodyFontSize: getComputedStyle(document.body).fontSize
  }));
}

test('native 100–200% browser zoom preserves reading and controls across all skins', async ({ request }, testInfo) => {
  test.setTimeout(120000);
  await request.post('/fixture/reset');
  const extension = fileURLToPath(new URL('./extension', import.meta.url));
  // viewport:null is intentional: no device metrics or CSS zoom emulation.
  const context = await chromium.launchPersistentContext('', {
    channel: 'chromium', headless: testInfo.project.use.headless ?? true,
    viewport: null, locale: 'zh-CN', timezoneId: 'Asia/Shanghai', reducedMotion: 'reduce',
    args: [`--disable-extensions-except=${extension}`, `--load-extension=${extension}`, '--window-size=1280,900']
  });
  const rows = [], errors = [], external = [];
  let completed = false, userAgent = '';
  await context.tracing.start({ screenshots: true, snapshots: true });
  try {
    let [worker] = context.serviceWorkers();
    if (!worker) worker = await context.waitForEvent('serviceworker');
    const page = context.pages()[0] || await context.newPage();
    page.on('pageerror', error => errors.push(error.message));
    await page.route('**/*', route => {
      if (new URL(route.request().url()).origin !== origin) { external.push(route.request().url()); return route.abort(); }
      return route.continue();
    });
    await page.clock.setFixedTime(new Date('2026-09-12T00:10:00Z'));
    await page.goto(origin);
    await expect(page.locator('html')).toHaveClass(/hydrated/);
    await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
    userAgent = await page.evaluate(() => navigator.userAgent);
    await page.evaluate(() => { window.zoomDocumentMarker = 'same reader document'; });
    expect(await setZoom(worker, 1)).toBe(1);
    const baseline = await metrics(page);

    for (const skin of ['slate', 'paper', 'grove', 'terminal']) {
      for (const mode of ['light', 'dark']) {
        await page.locator('#skin').selectOption(skin);
        await page.locator('#mode').selectOption(mode);
        for (const factor of [1, 1.25, 1.5, 1.75, 2]) {
          const nativeFactor = await setZoom(worker, factor);
          expect(nativeFactor).toBeCloseTo(factor, 2);
          await expect.poll(async () => (await metrics(page)).pixelRatio).toBeCloseTo(baseline.pixelRatio * factor, 2);
          const current = await metrics(page);
          rows.push({ skin, mode, nativeFactor, ...current });
          expect(current.outerWidth).toBe(baseline.outerWidth);
          expect(Math.abs(current.innerWidth * factor - baseline.innerWidth)).toBeLessThanOrEqual(3);
          expect(current.scrollWidth).toBeLessThanOrEqual(current.innerWidth + 1);
          expect(current.pinchScale).toBe(1);
          expect(['1', 'normal']).toContain(current.cssZoom);
          expect(current.bodyFontSize).toBe(baseline.bodyFontSize);
          await expect(page.locator('#sources > article')).toHaveCount(3);
          expect(await page.locator('.article-title').evaluateAll(elements => elements.filter(el => el.getClientRects().length).every(el => el.scrollWidth <= el.clientWidth + 1 && el.scrollHeight <= el.clientHeight + 1))).toBe(true);
        }

        // Exercise the actual controls at 200%, not just screenshot dimensions.
        await page.locator('#skin').focus();
        for (const id of ['mode', 'density', 'search', 'clear-search', 'source-filter', 'refresh-view']) {
          await page.keyboard.press('Tab');
          await expect(page.locator(`#${id}`)).toBeFocused();
        }
        await page.getByRole('searchbox').fill('没有这篇文章');
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
        await page.getByRole('button', { name: '清空', exact: true }).click();
        const card = page.getByRole('article', { name: '技术周刊', exact: true });
        const more = card.locator('[data-action="more"]');
        if (await more.getAttribute('aria-expanded') !== 'true') { await more.focus(); await page.keyboard.press('Enter'); }
        await expect(card.getByRole('link', { name: /技术周刊：第 12 篇/ })).toBeVisible();
        await expect(card.getByRole('list')).toBeVisible();
        await page.evaluate(() => scrollTo(0, 0));
        await testInfo.attach(`${skin}-${mode}-native-zoom-200`, { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' });
      }
    }
    await request.post('/fixture/new');
    await expect(page.getByRole('button', { name: '应用更新' })).toBeVisible();
    await page.getByRole('button', { name: '应用更新' }).focus(); await page.keyboard.press('Enter');
    await expect(page.getByRole('link', { name: '新文章：可靠的实时阅读' })).toBeVisible();
    await expect(page.locator('#refresh-view')).toBeFocused();
    expect(await page.evaluate(() => window.zoomDocumentMarker)).toBe('same reader document');
    await testInfo.attach('native-zoom-aria-snapshot', { body: Buffer.from(await page.locator('body').ariaSnapshot()), contentType: 'text/plain' });
    expect(rows).toHaveLength(40);
    expect(errors).toEqual([]); expect(external).toEqual([]);
    completed = true;
  } finally {
    await testInfo.attach('native-zoom-metrics', { body: Buffer.from(JSON.stringify({ userAgent, mechanism: 'chrome.tabs.setZoom/getZoom; automatic, per-tab; no viewport emulation', completed, rows }, null, 2)), contentType: 'application/json' });
    await context.tracing.stop(completed ? {} : { path: testInfo.outputPath('native-zoom-trace.zip') });
    await context.close();
  }
});

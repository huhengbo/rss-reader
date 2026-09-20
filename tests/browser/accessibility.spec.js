import { test, expect } from '@playwright/test';

test('named native controls retain keyboard order and recover focus after source removal', async ({ page, request }, testInfo) => {
  await request.post('/fixture/reset');
  await page.goto('/');
  await expect(page.locator('html')).toHaveClass(/hydrated/);
  await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN');
  await expect(page.getByRole('main')).toHaveCount(1);
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: '跳到订阅内容' })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('main')).toBeFocused();
  for (const id of ['search', 'clear-search', 'refresh-view', 'skin', 'mode', 'density', 'source-filter']) {
    await page.keyboard.press('Tab');
    const control = page.locator(`#${id}`);
    await expect(control).toBeFocused();
    await expect(control).toHaveAccessibleName(/\S/);
  }
  const card = page.getByRole('article', { name: '技术周刊', exact: true });
  await expect(card.getByRole('heading', { level: 2 })).toBeVisible();
  const list = card.getByRole('list');
  await expect(list.getByRole('listitem')).toHaveCount(8);
  const more = card.locator('[data-action="more"]');
  await expect(more).toHaveAttribute('aria-expanded', 'false');
  await expect(more).toHaveAttribute('aria-controls', await list.getAttribute('id'));
  const toggle = card.getByRole('button', { name: '折叠 技术周刊', exact: true });
  const controlled = await toggle.getAttribute('aria-controls');
  await expect(page.locator(`[id="${controlled}"]`)).toBeVisible();
  await testInfo.attach('native-roles-before-removal', { body: Buffer.from(await card.ariaSnapshot()), contentType: 'text/plain' });
  await toggle.focus();
  await request.post('/fixture/delete');
  await expect(card).toHaveCount(0);
  await expect(page.getByRole('main')).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(page.locator('#search')).toBeFocused();
});

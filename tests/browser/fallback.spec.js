import { test, expect } from '@playwright/test';

test.beforeEach(async ({ request }) => {
  await request.post('/fixture/reset');
});

for (const script of ['app.js', 'model.js']) {
  test(`all server-rendered articles remain readable when ${script} fails to load`, async ({ page }) => {
    // JavaScript stays enabled: the first-paint preferences script still runs.
    // This is distinct from the existing javaScriptEnabled:false regression.
    await page.route(`**/static/${script}`, route => route.abort());
    await page.goto('/');
    await expect(page.locator('html')).not.toHaveClass(/hydrated/);
    await expect(page.getByRole('heading', { name: '技术周刊', exact: true })).toBeVisible();
    const lastArticle = page.getByRole('link', { name: /技术周刊：第 12 篇/, includeHidden: true });
    await expect(lastArticle).toHaveAttribute('href', /^https?:\/\//);
    await expect(lastArticle).toBeVisible();
    await expect(page.locator('[data-action="more"]').first()).toBeHidden();
  });
}

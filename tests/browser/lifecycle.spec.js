import { test, expect } from '@playwright/test';

test.beforeEach(async ({ request }) => {
  await request.post('/fixture/reset');
});

test('successive one-shot connections deliver their first snapshot without a retry', async ({ page, request }) => {
  await request.post('/fixture/zero');
  let sockets = 0;
  page.on('websocket', () => sockets++);
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  for (let i = 0; i < 6; i++) {
    await page.goto('/');
    await expect(page.locator('html')).toHaveClass(/hydrated/);
    await expect(page.locator('#connection')).toHaveAttribute('data-state', 'snapshot');
    await expect(page.locator('#sources > article')).toHaveCount(3);
    expect(sockets).toBe(i + 1);
  }
  expect(errors).toEqual([]);
});

test('server restart closes the old stream and opens one replacement without page reload', async ({ page, request }) => {
  const snapshot = await (await request.get('/api/v1/snapshot')).json();
  const streams = [];
  let navigations = 0;
  page.on('framenavigated', frame => { if (frame === page.mainFrame()) navigations++; });
  await page.routeWebSocket('**/ws?v=1', ws => {
    streams.push(ws);
    ws.send(JSON.stringify(snapshot));
  });
  await page.goto('/');
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  await page.getByRole('searchbox').fill('技术');
  streams[0].close({ code: 1012, reason: 'service restart fixture' });
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'reconnecting');
  await expect.poll(() => streams.length).toBe(2);
  await expect(page.locator('#connection')).toHaveAttribute('data-state', 'connected');
  await expect(page.getByRole('searchbox')).toHaveValue('技术');
  expect(navigations).toBe(1);
});

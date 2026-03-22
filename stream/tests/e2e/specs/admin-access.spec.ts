import { test, expect } from '@playwright/test';
import { loginAs, waitForAuth } from '../helpers/auth';

/**
 * 管理画面のアクセス制御
 *
 * - sys-admin  → Admin Dashboard が表示される
 * - nfs-editor → Admin Dashboard が表示される (content:write 権限)
 * - nfs-viewer → Admin リンクが Navbar ドロップダウンに表示されない
 * - sys-admin  → Admin リンクが Navbar ドロップダウンに表示される
 */

test('sys-admin は Admin Dashboard にアクセスできる', async ({ browser }) => {
  const ctx = await browser.newContext();
  await loginAs(ctx, 'sys-admin');
  const page = await ctx.newPage();

  await page.goto('/admin');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('h1')).toContainText('Admin Dashboard');
  await expect(page.locator('table')).toBeVisible();
  await ctx.close();
});

test('nfs-editor は Admin Dashboard にアクセスできる', async ({ browser }) => {
  const ctx = await browser.newContext();
  await loginAs(ctx, 'nfs-editor');
  const page = await ctx.newPage();

  await page.goto('/admin');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('h1')).toContainText('Admin Dashboard');
  await ctx.close();
});

test('nfs-viewer の Navbar ドロップダウンに Admin リンクが表示されない', async ({ browser }) => {
  const ctx = await browser.newContext();
  await loginAs(ctx, 'nfs-viewer');
  const page = await ctx.newPage();

  await page.goto('/');
  await page.waitForLoadState('networkidle');
  await waitForAuth(page);

  // ドロップダウンをホバーで開いて確認
  // ユーザープロフィールボタン (w-10 h-10 rounded-full bg-surface)
  await page.locator('nav button.bg-surface').hover();
  await page.waitForTimeout(300);

  await expect(page.locator('a[href="/admin"]')).toHaveCount(0);
  await ctx.close();
});

test('sys-admin の Navbar ドロップダウンに Admin リンクが表示される', async ({ browser }) => {
  const ctx = await browser.newContext();
  await loginAs(ctx, 'sys-admin');
  const page = await ctx.newPage();

  await page.goto('/');
  await page.waitForLoadState('networkidle');
  await waitForAuth(page);

  // ドロップダウンをホバーで開いて確認
  // ユーザープロフィールボタン (w-10 h-10 rounded-full bg-surface)
  await page.locator('nav button.bg-surface').hover();
  await expect(page.locator('a[href="/admin"]')).toBeVisible({ timeout: 5000 });
  await ctx.close();
});

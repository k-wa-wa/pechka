import { test, expect, type Page } from '@playwright/test';
import axios from 'axios';
import { loginAs, getCredentials } from '../helpers/auth';

const BASE = process.env.BASE_URL || 'http://localhost:8000';
const METADATA_URL = `${BASE}/api/metadata/v1`;

/**
 * 編集モーダルの動作検証
 *
 * - タブ切り替え (Videos / 360° VR / Gallery / E-Book)
 * - Edit ボタンでモーダルが開く
 * - visibility: Public / Group Only の切り替え
 * - Group Only 選択時にグループ一覧が表示される
 * - Public に戻すとグループ一覧が非表示になる
 * - Cancel でモーダルが閉じる
 * - Edit All Data で bulk edit モードに入れる
 * - visibility を group_only に変更して保存できる
 */

/** admin ページを開き、コンテンツテーブルがレンダリング完了するまで待つ */
async function openAdminPage(page: Page) {
  await page.goto('/admin');
  await page.waitForLoadState('networkidle');
  await page.waitForSelector('table', { timeout: 15000 });
  // Loading... が消えるまで待つ（コンテンツ行 or "No content found" が出る）
  await page.waitForFunction(
    () => {
      const cells = Array.from(document.querySelectorAll('td'));
      return cells.length > 0 && !cells.some(td => td.textContent?.includes('Loading...'));
    },
    { timeout: 15000 },
  );
}

test.describe('管理画面 編集モーダル', () => {
  let contentTitle: string;

  test.beforeAll(async () => {
    const { appToken } = await getCredentials('sys-admin');
    const headers = { Authorization: `Bearer ${appToken}` };
    contentTitle = `E2Eテスト用動画-${Date.now()}`;
    await axios.post(
      `${METADATA_URL}/admin/metadata/contents`,
      {
        content_type: 'video',
        title: contentTitle,
        description: 'E2E modal test',
        video_details: { is_360: false, duration_seconds: 10, director: 'E2E' },
      },
      { headers },
    );
  });

  test('コンテンツタブを切り替えられる', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    for (const label of ['360° VR', 'Gallery', 'E-Book', 'Videos'] as const) {
      await page.locator('button', { hasText: label }).first().click();
      await expect(
        page.locator('button', { hasText: label }).first()
      ).toHaveClass(/bg-white/);
    }

    await ctx.close();
  });

  test('Edit ボタンでモーダルが開く', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    const editBtn = page.locator('button[title="Edit"]').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();

    await expect(page.getByText('Access Control')).toBeVisible();
    await expect(page.locator('button', { hasText: 'Public' }).first()).toBeVisible();
    await expect(page.locator('button', { hasText: 'Group Only' }).first()).toBeVisible();

    await ctx.close();
  });

  test('Group Only に切り替えるとグループ一覧が表示される', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    await page.locator('button[title="Edit"]').first().click();
    await expect(page.getByText('Access Control')).toBeVisible({ timeout: 10000 });

    await page.locator('button', { hasText: 'Group Only' }).first().click();
    await expect(page.getByText('アクセスを許可するグループを選択')).toBeVisible();

    await ctx.close();
  });

  test('Public に戻すとグループ一覧が非表示になる', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    await page.locator('button[title="Edit"]').first().click();
    await expect(page.getByText('Access Control')).toBeVisible({ timeout: 10000 });

    await page.locator('button', { hasText: 'Group Only' }).first().click();
    await expect(page.getByText('アクセスを許可するグループを選択')).toBeVisible();

    await page.locator('button', { hasText: 'Public' }).first().click();
    await expect(page.getByText('アクセスを許可するグループを選択')).toHaveCount(0);

    await ctx.close();
  });

  test('Cancel でモーダルが閉じる', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    await page.locator('button[title="Edit"]').first().click();
    await expect(page.getByText('Access Control')).toBeVisible({ timeout: 10000 });

    await page.locator('button', { hasText: 'Cancel' }).click();
    await expect(page.getByText('Access Control')).toHaveCount(0);

    await ctx.close();
  });

  test('Edit All Data で bulk edit モードになる', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    await page.locator('button', { hasText: 'Edit All Data' }).click();
    await expect(page.locator('button', { hasText: 'Save Bulk Changes' })).toBeVisible();

    await ctx.close();
  });

  test('visibility を group_only に変更して保存できる', async ({ browser }) => {
    const ctx = await browser.newContext();
    await loginAs(ctx, 'sys-admin');
    const page = await ctx.newPage();
    await openAdminPage(page);

    // テーブル先頭のコンテンツを使用
    const editBtn = page.locator('button[title="Edit"]').first();
    await expect(editBtn).toBeVisible({ timeout: 10000 });
    await editBtn.click();

    await expect(page.getByText('Access Control')).toBeVisible({ timeout: 10000 });
    await page.locator('button', { hasText: 'Group Only' }).first().click();

    const groupBtn = page.locator('button', { hasText: 'nfs-admin' });
    if ((await groupBtn.count()) > 0) {
      await groupBtn.first().click();
    }

    await page.locator('button', { hasText: 'Save Changes' }).click();
    await expect(page.getByText('保存しました')).toBeVisible({ timeout: 10000 });

    await ctx.close();
  });
});

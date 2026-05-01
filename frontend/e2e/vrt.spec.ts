import { test, expect } from '@playwright/test'

async function waitForPageReady(page: import('@playwright/test').Page) {
  // Wait for next/dynamic loading placeholders to disappear (ssr:false components)
  await page.waitForFunction(
    () => !document.body.innerText.includes('読み込み中...') && !document.body.innerText.includes('VRビューアを読み込み中...'),
    { timeout: 10_000 }
  )
  // Small settle time for animations
  await page.waitForTimeout(300)
}

test.describe('Visual Regression Tests', () => {
  test('home page - content list', async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('home.png', { fullPage: true })
  })

  test('home page - video filter', async ({ page }) => {
    await page.goto('/?type=video')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('home-video-filter.png', { fullPage: true })
  })

  test('admin page', async ({ page }) => {
    await page.goto('/admin')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('admin.png', { fullPage: true })
  })

  test('content detail - video', async ({ page }) => {
    await page.goto('/contents/vid001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('content-video.png', { fullPage: true })
  })

  test('content detail - vr360', async ({ page }) => {
    await page.goto('/contents/vr001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('content-vr360.png', { fullPage: true })
  })
})

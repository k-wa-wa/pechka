import { test, expect } from '@playwright/test'

const SCREENSHOT_OPTIONS = { fullPage: true, maxDiffPixelRatio: 0.02 }

async function waitForPageReady(page: import('@playwright/test').Page) {
  // Wait for Next.js dynamic import placeholders to disappear
  await page.waitForFunction(
    () =>
      !document.body.innerText.includes('読み込み中...') &&
      !document.body.innerText.includes('VRビューアを読み込み中...'),
    { timeout: 15_000 }
  )
  // Wait for A-Frame scene to finish loading if present
  await page.waitForFunction(
    () => {
      const scene = document.querySelector('a-scene')
      if (!scene) return true
      return scene.hasAttribute('loaded')
    },
    { timeout: 15_000 }
  )
  // Wait for network to settle (images, fonts, etc.)
  await page.waitForLoadState('networkidle', { timeout: 15_000 })
  // Small settle time for CSS transitions
  await page.waitForTimeout(300)
}

test.describe('Visual Regression Tests', () => {
  test('home page - content list', async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('home.png', SCREENSHOT_OPTIONS)
  })

  test('home page - video filter', async ({ page }) => {
    await page.goto('/?type=video')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('home-video-filter.png', SCREENSHOT_OPTIONS)
  })

  test('admin page', async ({ page }) => {
    await page.goto('/admin')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('admin.png', SCREENSHOT_OPTIONS)
  })

  test('content detail - video', async ({ page }) => {
    await page.goto('/contents/vid001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('content-video.png', SCREENSHOT_OPTIONS)
  })

  test('content detail - vr360', async ({ page }) => {
    await page.goto('/contents/vr001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await waitForPageReady(page)
    await expect(page).toHaveScreenshot('content-vr360.png', SCREENSHOT_OPTIONS)
  })
})

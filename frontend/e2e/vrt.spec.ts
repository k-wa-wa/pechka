import { test, expect } from '@playwright/test'

test.describe('Visual Regression Tests', () => {
  test('home page - content list', async ({ page }) => {
    await page.goto('/')
    // Wait for content to render
    await page.waitForSelector('h1', { timeout: 10_000 })
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('home.png', { fullPage: true })
  })

  test('home page - video filter', async ({ page }) => {
    await page.goto('/?type=video')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('home-video-filter.png', { fullPage: true })
  })

  test('admin page', async ({ page }) => {
    await page.goto('/admin')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('admin.png', { fullPage: true })
  })

  test('content detail - video', async ({ page }) => {
    await page.goto('/contents/vid001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('content-video.png', { fullPage: true })
  })

  test('content detail - vr360', async ({ page }) => {
    await page.goto('/contents/vr001')
    await page.waitForSelector('h1', { timeout: 10_000 })
    await page.waitForTimeout(1000)
    await expect(page).toHaveScreenshot('content-vr360.png', { fullPage: true })
  })
})

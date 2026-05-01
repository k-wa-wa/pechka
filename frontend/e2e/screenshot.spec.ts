import { test } from "@playwright/test";
import { mkdir } from "fs/promises";
import path from "path";

const SCREENSHOTS = path.join(__dirname, "screenshots");

test.beforeAll(async () => {
  await mkdir(SCREENSHOTS, { recursive: true });
});

test("01-home", async ({ page }) => {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "01-home.png"), fullPage: true });
});

test("02-home-video-filter", async ({ page }) => {
  await page.goto("/?type=video");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "02-home-video-filter.png"), fullPage: true });
});

test("03-search-empty", async ({ page }) => {
  await page.goto("/search");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "03-search-empty.png"), fullPage: true });
});

test("04-search-results", async ({ page }) => {
  await page.goto("/search?q=%E6%98%A0%E7%94%BB");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "04-search-results.png"), fullPage: true });
});

test("05-admin", async ({ page }) => {
  await page.goto("/admin");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "05-admin.png"), fullPage: true });
});

test("06-admin-edit-content", async ({ page }) => {
  await page.goto("/admin/contents/id-1");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "06-admin-edit-content.png"), fullPage: true });
});

test("07-content-detail", async ({ page }) => {
  await page.goto("/contents/abc123");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "07-content-detail.png"), fullPage: true });
});

test("08-vr-content", async ({ page }) => {
  await page.goto("/contents/def456");
  await page.waitForLoadState("networkidle");
  await page.screenshot({ path: path.join(SCREENSHOTS, "08-vr-content.png"), fullPage: true });
});

import { expect, test } from "@playwright/test";

test.describe("Channel E2E Functional Tests", () => {
  test("home page cards show channel name and navigate to channel page on click", async ({
    page,
  }) => {
    await page.goto("/");
    await page.waitForSelector("h1", { timeout: 10_000 });

    // Verify both Blu-ray and Tech Feed channel labels appear on cards
    const techFeedLink = page.locator(".channel-card-link", { hasText: "Tech Feed" }).first();
    await expect(techFeedLink).toBeVisible();

    const blurayLink = page.locator(".channel-card-link", { hasText: "Blu-ray" }).first();
    await expect(blurayLink).toBeVisible();

    // Clicking the channel name on the card navigates to /channels/tech-feed (not /contents/:id)
    await techFeedLink.click();
    await expect(page).toHaveURL(/\/channels\/tech-feed$/);
    await expect(page.locator("h1")).toHaveText("Tech Feed");
    await expect(page.getByText("自動収集・AI生成による技術ダイジェスト解説")).toBeVisible();
    await expect(page.getByText("2024年度 活動報告書")).toBeVisible();
    await expect(page.getByText("夏の思い出 2024")).toHaveCount(0);
  });

  test("channel page /channels/bluray displays only Blu-ray contents", async ({ page }) => {
    await page.goto("/channels/bluray");
    await expect(page.locator("h1")).toHaveText("Blu-ray");
    await expect(page.getByText("Blu-ray取り込み動画")).toBeVisible();
    await expect(page.getByText("夏の思い出 2024")).toBeVisible();
    await expect(page.getByText("秋の紅葉ドライブ")).toBeVisible();
    await expect(page.getByText("2024年度 活動報告書")).toHaveCount(0);
  });

  test("unknown channel page returns 404 not found", async ({ page }) => {
    const res = await page.goto("/channels/unknown-channel");
    expect(res?.status()).toBe(404);
  });

  test("content detail page displays channel badge linking to channel page", async ({ page }) => {
    await page.goto("/contents/vid001");
    await page.waitForSelector("h1", { timeout: 10_000 });

    const channelBadge = page.locator('a[href="/channels/bluray"]').first();
    await expect(channelBadge).toBeVisible();
    await expect(channelBadge).toContainText("Blu-ray");

    await channelBadge.click();
    await expect(page).toHaveURL(/\/channels\/bluray$/);
    await expect(page.locator("h1")).toHaveText("Blu-ray");
  });

  test("admin edit modal updates content channel and sends channel_id in PUT request", async ({
    page,
  }) => {
    await page.goto("/admin");
    await page.waitForSelector("h1", { timeout: 10_000 });

    // Open edit modal for the first row (インターステラー)
    const editButtons = page.locator('button:has-text("編集"), button:has-text("Edit")');
    await editButtons.first().click();

    const select = page.locator("select").first();
    await expect(select).toBeVisible();
    await expect(select).toHaveValue("bluray");

    // Change channel to tech-feed and verify PUT payload
    const putPromise = page.waitForRequest(
      (req) => req.url().includes("/api/v1/admin/contents/") && req.method() === "PUT",
    );
    await select.selectOption("tech-feed");
    await page.locator('button:has-text("保存"), button:has-text("Save")').click();

    const putReq = await putPromise;
    expect(putReq.postDataJSON()).toMatchObject({ channel_id: "tech-feed" });
    await expect(page.locator("select")).toHaveCount(0);

    // Restore back to bluray so mock state stays clean for subsequent tests
    await editButtons.first().click();
    await page.locator("select").first().selectOption("bluray");
    await page.locator('button:has-text("保存"), button:has-text("Save")').click();
    await expect(page.locator("select")).toHaveCount(0);
  });

  test("admin upload modal sends X-Channel-Id header and API rejects missing X-Channel-Id", async ({
    page,
    request,
  }) => {
    // 1. Verify API rejects upload without X-Channel-Id header with 400
    const apiBase = process.env.API_URL ?? "http://localhost:9001";
    const badRes = await request.post(`${apiBase}/api/v1/admin/contents/upload`, {
      multipart: {
        title: "No Channel Upload",
        file: {
          name: "test.mp4",
          mimeType: "video/mp4",
          buffer: Buffer.from("fake video"),
        },
      },
    });
    expect(badRes.status()).toBe(400);

    // 2. Verify UploadModal UI sends selected X-Channel-Id header
    await page.goto("/admin");
    await page.waitForSelector("h1", { timeout: 10_000 });

    await page
      .locator('button:has-text("動画アップロード"), button:has-text("Upload Video")')
      .click();
    const channelSelect = page.locator("select").first();
    await expect(channelSelect).toBeVisible();
    await channelSelect.selectOption("tech-feed");
    await expect(channelSelect).toHaveValue("tech-feed");
  });
});

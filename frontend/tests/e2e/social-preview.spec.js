import { test, expect } from "@playwright/test";

test("LinkedIn preview metadata and JPEG are accessible without running React", async ({
  browser,
  baseURL,
}) => {
  const context = await browser.newContext({
    baseURL,
    javaScriptEnabled: false,
    userAgent: "LinkedInBot/1.0",
  });
  try {
    const page = await context.newPage();
    const response = await page.goto("/");
    expect(response.status()).toBe(200);
    await expect(page.locator("#root")).toBeEmpty();

    const property = (name) => page.locator(`meta[property="${name}"]`);
    const named = (name) => page.locator(`meta[name="${name}"]`);
    const imageUrl = "https://survive.phfaria.com/social-preview.jpg";
    await expect(property("og:type")).toHaveAttribute("content", "website");
    await expect(property("og:url")).toHaveAttribute(
      "content",
      "https://survive.phfaria.com/",
    );
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
      "href",
      "https://survive.phfaria.com/",
    );
    await expect(property("og:title")).toHaveAttribute(
      "content",
      /SURVIVE THE DESTRUCTION/,
    );
    await expect(property("og:description")).toHaveAttribute(
      "content",
      /Go, WebSocket and React/,
    );
    await expect(property("og:image")).toHaveAttribute("content", imageUrl);
    await expect(property("og:image:secure_url")).toHaveAttribute(
      "content",
      imageUrl,
    );
    await expect(property("og:image:type")).toHaveAttribute(
      "content",
      "image/jpeg",
    );
    await expect(property("og:image:width")).toHaveAttribute("content", "1200");
    await expect(property("og:image:height")).toHaveAttribute("content", "630");
    await expect(property("og:image:alt")).toHaveAttribute(
      "content",
      /floating islands/,
    );
    await expect(named("twitter:card")).toHaveAttribute(
      "content",
      "summary_large_image",
    );
    await expect(named("twitter:image")).toHaveAttribute("content", imageUrl);
    expect(await named("twitter:title").getAttribute("content")).toBe(
      await property("og:title").getAttribute("content"),
    );
    expect(await named("twitter:description").getAttribute("content")).toBe(
      await named("description").getAttribute("content"),
    );

    // Exercita o asset copiado pelo build, não a URL remota ainda sem deploy.
    const imagePath = new URL(imageUrl).pathname;
    const image = await context.request.get(imagePath);
    expect(image.status()).toBe(200);
    expect(image.headers()["content-type"]).toMatch(/^image\/jpeg/);
    const bytes = await image.body();
    expect([...bytes.subarray(0, 3)]).toEqual([0xff, 0xd8, 0xff]);
    expect(bytes.length).toBeLessThan(5 * 1024 * 1024);
    await page.goto(imagePath);
    await expect
      .poll(() =>
        page
          .locator("img")
          .evaluate((element) => ({
            width: element.naturalWidth,
            height: element.naturalHeight,
          })),
      )
      .toEqual({ width: 1200, height: 630 });
  } finally {
    await context.close();
  }
});

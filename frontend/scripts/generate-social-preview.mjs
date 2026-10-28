import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";

// Geração manual: a Vercel apenas copia o JPEG versionado em public/.
// Não exige navegador ou renderização dinâmica durante o deploy/scraping.
const artwork = await readFile(
  new URL("../assets/carregamento1.jpeg", import.meta.url),
);
const output = new URL("../public/social-preview.jpg", import.meta.url);
const browser = await chromium.launch();

try {
  const page = await browser.newPage({
    viewport: { width: 1200, height: 630 },
    deviceScaleFactor: 1,
    reducedMotion: "reduce",
  });
  await page.setContent(`<!doctype html>
    <html lang="en">
      <head><meta charset="UTF-8"><style>
        * { box-sizing: border-box; }
        html, body { margin: 0; width: 1200px; height: 630px; overflow: hidden; }
        body { background: #faf5e9; color: #303a27; font-family: Arial, sans-serif; }
        main { display: grid; grid-template-columns: 624px 576px; height: 630px; }
        .copy { padding: 45px 44px 36px 48px; display: flex; flex-direction: column; }
        .eyebrow { display: flex; align-items: center; gap: 12px; font-size: 12px; font-weight: 700; letter-spacing: 2px; }
        .orb { height: 28px; width: 28px; border-radius: 50%; background: radial-gradient(circle at 35% 25%, #ffe3a3, #edaf34 60%, #c17e19); box-shadow: 0 3px 0 #947142; }
        h1 { font-size: 68px; line-height: 1.03; letter-spacing: -2.7px; font-weight: 900; margin: 31px 0 23px; }
        .tagline { margin: 0; font-family: Georgia, serif; font-size: 37px; line-height: 1.2; font-style: italic; color: #63733d; }
        .description { max-width: 445px; margin: 18px 0 0; font-size: 18px; line-height: 1.6; color: #62664f; }
        .tags { display: flex; gap: 10px; margin-top: 25px; }
        .tags span { font-size: 12px; border: 1px solid #a9b497; border-radius: 5px; padding: 8px 12px; }
        .domain { margin-top: auto; padding-top: 20px; font-size: 15px; letter-spacing: 1px; color: #485a32; }
        .art { margin: 23px 23px 23px 0; overflow: hidden; border: 7px solid #fffdf7; border-radius: 155px 155px 15px 15px; box-shadow: 0 12px 24px #4a553018; }
        img { display: block; width: 100%; height: 100%; object-fit: cover; object-position: center; }
      </style></head>
      <body><main>
        <section class="copy">
          <div class="eyebrow"><span class="orb"></span>REAL-TIME MULTIPLAYER</div>
          <h1>SURVIVE THE<br>DESTRUCTION</h1>
          <p class="tagline">The floor disappears.<br>You stay.</p>
          <p class="description">A survival game on floating islands.<br>Built to learn. Made to play.</p>
          <div class="tags"><span>Go</span><span>WebSocket</span><span>React + Canvas</span></div>
          <div class="domain">survive.phfaria.com</div>
        </section>
        <figure class="art"><img src="data:image/jpeg;base64,${artwork.toString("base64")}" alt="Blue character on a crumbling floating island"></figure>
      </main></body>
    </html>`);
  await page.locator("img").evaluate((image) => image.decode());
  await page.screenshot({
    path: fileURLToPath(output),
    type: "jpeg",
    quality: 88,
  });
  console.info("Generated public/social-preview.jpg (1200 × 630).");
} finally {
  await browser.close();
}

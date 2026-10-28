import { test, expect } from "@playwright/test";

async function observe(page) {
  const record = {
    state: null,
    init: null,
    falling: false,
    roundOver: false,
    moved: false,
    rounds: new Set(),
  };
  page.on("websocket", (socket) =>
    socket.on("framereceived", ({ payload }) => {
      let data;
      try {
        data = JSON.parse(payload);
      } catch {
        return;
      }
      if (data.type === "init") record.init = data;
      if (!data.players) return;
      record.state = data;
      record.rounds.add(data.round);
      record.falling ||= Object.values(data.arena_tiles).some(
        (tile) => tile.is_falling,
      );
      record.roundOver ||= data.round_over;
      record.moved ||= Object.values(data.players).some(
        (player) => player.name.startsWith("Demo Pedro") && player.vx > 0,
      );
    }),
  );
  await page.addInitScript(() => {
    const NativeSocket = window.WebSocket;
    window.WebSocket = class extends NativeSocket {
      constructor(...args) {
        super(...args);
        window.demoSocket = this;
      }
    };
  });
  return record;
}

async function enter(page, name) {
  const health = await page.request.get("http://127.0.0.1:18080/api/health", { timeout: 5000 });
  await expect(health, "Inicie o backend de teste conforme docs/demonstracao.md").toBeOK();
  await page.goto("/");
  await page.getByRole("button", { name: "Jogar agora" }).first().click();
  await page.getByLabel("Escolha seu nome").fill(name);
  await page.getByRole("button", { name: "Entrar na arena" }).click();
  await expect(page.locator("canvas")).toBeVisible();
}

test("home and personalization work without backend, in Portuguese and English", async ({
  page,
}) => {
  let requests = 0;
  await page.route("**/api/**", (route) => {
    requests++;
    return route.abort();
  });
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "O chão desaparece",
  );
  await page.screenshot({
    path: "../docs/evidence/home-desktop.png",
    fullPage: true,
  });
  await page
    .getByRole("heading", {
      name: "Um laboratório de multiplayer em tempo real.",
    })
    .scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../docs/evidence/home-projeto.png" });
  await page.getByRole("button", { name: "Switch to English" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "The floor disappears",
  );
  await page.getByRole("button", { name: "Mudar para português" }).click();
  await page.getByRole("button", { name: "Jogar agora" }).first().click();
  await expect(page.getByLabel("Escolha seu nome")).toBeFocused();
  await page.getByRole("button", { name: "Voltar ao início" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "O chão desaparece",
  );
  expect(requests).toBe(0);
});

test("mobile landscape runs a real match and accepts touch controls", async ({
  browser,
}) => {
  const context = await browser.newContext({
    viewport: { width: 844, height: 390 },
    isMobile: true,
    hasTouch: true,
    locale: "pt-BR",
  });
  const page = await context.newPage();
  const state = await observe(page);
  try {
    await enter(page, "Demo Mobile");
    await expect.poll(() => state.state?.room_players).toBe(1);
    const right = page.getByRole("button", { name: "Mover para a direita" });
    await expect(right).toBeVisible();
    const bounds = await right.boundingBox();
    const cdp = await context.newCDPSession(page);
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchStart",
      touchPoints: [
        { x: bounds.x + bounds.width / 2, y: bounds.y + bounds.height / 2 },
      ],
    });
    await expect
      .poll(() =>
        Object.values(state.state?.players || {}).some((p) => p.vx > 0),
      )
      .toBe(true);
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchEnd",
      touchPoints: [],
    });
    await page.screenshot({ path: "../docs/evidence/partida-mobile.png" });
    await page.getByRole("button", { name: "SAIR", exact: true }).tap();
    await expect(page.getByRole("heading", { level: 1 })).toContainText(
      "O chão desaparece",
    );
  } finally {
    await context.close();
  }
});

test("mobile home scrolls in portrait and personalization fits the viewport", async ({
  browser,
}) => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
    locale: "pt-BR",
  });
  const page = await context.newPage();
  try {
    await page.goto("/");
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    expect(
      await page
        .locator(".home-scroll")
        .evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);
    await page.screenshot({
      path: "../docs/evidence/home-mobile.png",
      fullPage: true,
    });
    await page.getByRole("button", { name: "Jogar agora" }).first().click();
    await expect(
      page.getByRole("button", { name: "Entrar na arena" }),
    ).toBeInViewport();
    await page.screenshot({
      path: "../docs/evidence/personalizacao-mobile.png",
    });
  } finally {
    await context.close();
  }
});

test("two real players share physics, round lifecycle, scores, reconnection and exit", async ({
  browser,
  request,
}) => {
  test.setTimeout(90000);
  const runId = Date.now().toString().slice(-6);
  const pedro = `Demo Pedro ${runId}`;
  const ana = `Demo Ana ${runId}`;
  const a = await browser.newContext({
    locale: "pt-BR",
    viewport: { width: 1440, height: 1000 },
  });
  const b = await browser.newContext({
    locale: "pt-BR",
    viewport: { width: 1440, height: 1000 },
  });
  const pa = await a.newPage(),
    pb = await b.newPage();
  const errors = [];
  pa.on("pageerror", (error) => errors.push(error.message));
  pb.on("pageerror", (error) => errors.push(error.message));
  const ra = await observe(pa),
    rb = await observe(pb);
  try {
    await enter(pa, pedro);
    await enter(pb, ana);
    await expect.poll(() => rb.state?.room_players).toBe(2);
    expect(ra.init.room_id).toBe(rb.init.room_id);
    await expect
      .poll(() =>
        Object.values(rb.state?.players || {})
          .map((p) => p.name)
          .sort(),
      )
      .toEqual([ana, pedro]);
    await pa.locator("canvas").click();
    await pa.keyboard.down("d");
    await expect.poll(() => rb.moved).toBe(true);
    await pa.keyboard.up("d");
    await pa.keyboard.press("Space");
    await pa.keyboard.press("Shift");
    await pa.screenshot({ path: "../docs/evidence/partida-desktop.png" });
    await expect.poll(() => rb.falling, { timeout: 12000 }).toBe(true);
    await pa.keyboard.down("d");
    await expect
      .poll(() => ra.roundOver || rb.roundOver, { timeout: 45000 })
      .toBe(true);
    await pa.keyboard.up("d");
    await expect
      .poll(() => rb.rounds.size, { timeout: 12000 })
      .toBeGreaterThan(1);
    await expect
      .poll(async () => {
        const response = await request.get("http://127.0.0.1:18080/api/scores");
        const scores = await response.json();
        return scores.some(
          (score) => score.name === pedro || score.name === ana,
        );
      })
      .toBe(true);
    const previous = ra.init.player_id;
    await pa.evaluate(() => window.demoSocket.close(1000, "demo reconnect"));
    await expect(pa.getByRole("status")).toContainText("Reconectando");
    await expect.poll(() => ra.init.player_id).not.toBe(previous);
    await expect.poll(() => ra.state?.room_players).toBe(2);
    await expect(pa.getByRole("status")).toHaveCount(0);
    await pa.getByRole("button", { name: "Sair da partida" }).click();
    await expect(pa.getByRole("heading", { level: 1 })).toContainText(
      "O chão desaparece",
    );
    await expect.poll(() => rb.state?.room_players).toBe(1);
    expect(errors).toEqual([]);
  } finally {
    await a.close();
    await b.close();
  }
});

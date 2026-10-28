import test from "node:test";
import assert from "node:assert/strict";
import { waitForBackend } from "../../src/network/backend_ready.js";

const config = { arena_width: 800, arena_height: 1000, tile_size: 100 };
test("readiness returns valid config only after successful health and config", async () => {
  const requests = [];
  const result = await waitForBackend(
    "http://test",
    new AbortController().signal,
    {
      fetcher: async (url) => {
        requests.push(url);
        return { ok: true, json: async () => config };
      },
    },
  );
  assert.deepEqual(result, config);
  assert.deepEqual(requests, [
    "http://test/api/health",
    "http://test/api/config",
  ]);
});

test("readiness retries failed health and invalid configuration", async () => {
  let healthCalls = 0,
    configCalls = 0;
  const result = await waitForBackend(
    "http://test",
    new AbortController().signal,
    {
      retryMs: 1,
      timeout: 2000,
      fetcher: async (url) =>
        url.endsWith("health")
          ? { ok: ++healthCalls > 1 }
          : { ok: true, json: async () => (++configCalls === 1 ? {} : config) },
    },
  );
  assert.deepEqual(result, config);
  assert.equal(healthCalls, 3);
});

test("leaving aborts readiness without additional requests", async () => {
  const ctrl = new AbortController();
  ctrl.abort();
  let called = false;
  await assert.rejects(
    waitForBackend("http://test", ctrl.signal, {
      fetcher: () => {
        called = true;
      },
    }),
  );
  assert.equal(called, false);
});

test("readiness has a finite timeout", async () => {
  await assert.rejects(
    waitForBackend("http://test", new AbortController().signal, { timeout: 0 }),
    /timeout/,
  );
});

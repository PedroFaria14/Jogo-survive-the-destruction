import { normalizeBaseUrl } from "./backend_config.js";

const wait = (ms, signal) =>
  new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(signal.reason);
      return;
    }
    const abort = () => {
      clearTimeout(timer);
      reject(signal.reason);
    };
    const timer = setTimeout(() => {
      signal.removeEventListener("abort", abort);
      resolve();
    }, ms);
    signal.addEventListener("abort", abort, { once: true });
  });

// Health e config usam o mesmo prazo e são cancelados ao sair da tela.
export async function waitForBackend(
  baseUrl,
  signal,
  { fetcher = fetch, timeout = 300000, retryMs = 4000 } = {},
) {
  const apiBaseUrl = normalizeBaseUrl(baseUrl);
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    signal.throwIfAborted();
    try {
      const requestSignal = AbortSignal.any([
        signal,
        AbortSignal.timeout(6000),
      ]);
      const health = await fetcher(`${apiBaseUrl}/api/health`, {
        signal: requestSignal,
      });
      if (!health.ok) throw new Error("Backend unavailable");
      const config = await fetcher(`${apiBaseUrl}/api/config`, {
        signal: requestSignal,
      });
      if (!config.ok) throw new Error("Configuration unavailable");
      const data = await config.json();
      if (
        !(data.arena_width > 0 && data.arena_height > 0 && data.tile_size > 0)
      )
        throw new Error("Invalid configuration");
      return data;
    } catch {
      signal.throwIfAborted();
    }
    await wait(retryMs, signal);
  }
  throw new Error("Backend readiness timeout");
}

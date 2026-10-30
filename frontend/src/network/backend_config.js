const DEFAULT_API_URL = "https://api.survive.phfaria.com";

export function normalizeBaseUrl(value) {
  return value.trim().replace(/\/+$/, "");
}

// Variáveis Vite são incorporadas no build. Se só a API mudar, o WebSocket
// acompanha o mesmo host/protocolo; um VITE_WS_URL explícito continua prioritário.
export function resolveBackendUrls(env = {}) {
  const apiBaseUrl =
    normalizeBaseUrl(env.VITE_API_URL || DEFAULT_API_URL) || DEFAULT_API_URL;
  const webSocketUrl =
    normalizeBaseUrl(env.VITE_WS_URL || "") ||
    `${apiBaseUrl.replace(/^https:/, "wss:").replace(/^http:/, "ws:")}/ws`;
  return { apiBaseUrl, webSocketUrl };
}

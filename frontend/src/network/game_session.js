// Transporte isolado de React e do canvas. Uma sessão possui no máximo um socket.
export function createGameSession({
  url,
  profile,
  input,
  onInit,
  onState,
  onStatus,
  Socket = WebSocket,
  schedule = setTimeout,
  cancel = clearTimeout,
  random = Math.random,
}) {
  let socket;
  let retry;
  let watchdog;
  let inputTimer;
  let stopped = false;
  let attempts = 0;
  let initialized = false;

  const send = (command) => {
    if (!stopped && socket?.readyState === 1) {
      socket.send(JSON.stringify(command));
      return true;
    }
    return false;
  };

  const connect = () => {
    if (stopped) return;
    initialized = false;
    onStatus(attempts ? "reconnecting" : "connecting");
    const current = new Socket(url);
    socket = current;
    // Também detecta upgrade travado e conexão aberta sem snapshots.
    watchdog = schedule(() => current.close(), 15000);
    current.onopen = () => {
      if (stopped || current !== socket) return;
      send({ type: "join", ...profile() });
      send({ type: "input", ...input() });
      const pulse = () => {
        if (stopped || current !== socket || current.readyState !== 1) return;
        send({ type: "input", ...input() });
        inputTimer = schedule(pulse, 100);
      };
      inputTimer = schedule(pulse, 100);
    };
    current.onmessage = ({ data: raw }) => {
      if (stopped || current !== socket) return;
      let data;
      try {
        data = JSON.parse(raw);
      } catch {
        return;
      }
      if (!data || typeof data !== "object") return;
      if (data.type === "init" && typeof data.player_id === "string") {
        initialized = true;
        onInit(data);
        return;
      }
      if (data.type || !initialized || !data.players || !data.arena_tiles)
        return;
      cancel(watchdog);
      attempts = 0;
      onStatus("connected");
      onState(data);
      watchdog = schedule(() => current.close(), 15000);
    };
    current.onerror = () => {
      /* onclose é a única origem de retries. */
    };
    current.onclose = () => {
      if (stopped || current !== socket) return;
      cancel(watchdog);
      onStatus("reconnecting");
      cancel(inputTimer);
      const delay = Math.min(1000 * 1.5 ** attempts++, 8000) + random() * 300;
      retry = schedule(connect, delay);
    };
  };
  connect();
  return {
    send,
    close() {
      stopped = true;
      cancel(retry);
      cancel(watchdog);
      cancel(inputTimer);
      if (socket) {
        socket.onopen =
          socket.onmessage =
          socket.onclose =
          socket.onerror =
            null;
        socket.close(1000, "leaving match");
      }
    },
  };
}

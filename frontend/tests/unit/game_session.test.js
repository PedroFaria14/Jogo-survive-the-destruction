import test from "node:test";
import assert from "node:assert/strict";
import { createGameSession } from "../../src/network/game_session.js";

function fixture() {
  const sockets = [],
    tasks = new Map(),
    states = [],
    statuses = [],
    inits = [];
  let seq = 0;
  class Socket {
    readyState = 0;
    sent = [];
    constructor() {
      sockets.push(this);
    }
    send(raw) {
      this.sent.push(JSON.parse(raw));
    }
    close() {
      this.readyState = 3;
      this.onclose?.();
    }
    open() {
      this.readyState = 1;
      this.onopen?.();
    }
    message(data) {
      this.onmessage?.({
        data: typeof data === "string" ? data : JSON.stringify(data),
      });
    }
  }
  const session = createGameSession({
    url: "ws://test/ws",
    profile: () => ({ name: "Pedro", color: "#123456" }),
    input: () => ({ left: true, right: false, jump: false, dash: false }),
    onInit: (x) => inits.push(x),
    onState: (x) => states.push(x),
    onStatus: (x) => statuses.push(x),
    Socket,
    schedule: (fn, delay) => {
      tasks.set(++seq, { fn, delay });
      return seq;
    },
    cancel: (id) => tasks.delete(id),
    random: () => 0,
  });
  const run = (delay) => {
    const task = [...tasks].find(([, value]) => value.delay === delay);
    assert.ok(task, `Missing timer ${delay}`);
    tasks.delete(task[0]);
    task[1].fn();
  };
  return { session, sockets, tasks, states, statuses, inits, run };
}

test("open sends profile and complete input; only accepts initialized snapshots", () => {
  const f = fixture(),
    s = f.sockets[0];
  s.open();
  assert.deepEqual(s.sent[0], {
    type: "join",
    name: "Pedro",
    color: "#123456",
  });
  assert.deepEqual(s.sent[1], {
    type: "input",
    left: true,
    right: false,
    jump: false,
    dash: false,
  });
  for (const data of ["broken", null, {}, { players: {}, arena_tiles: {} }])
    s.message(data);
  assert.equal(f.states.length, 0);
  s.message({ type: "init", player_id: "player_1" });
  s.message({ players: {}, arena_tiles: {}, round: 1 });
  assert.equal(f.inits[0].player_id, "player_1");
  assert.equal(f.states[0].round, 1);
  assert.equal(f.statuses.at(-1), "connected");
  f.session.close();
});

test("reconnect uses backoff, ignores stale sockets and requires a new init", () => {
  const f = fixture(),
    first = f.sockets[0];
  first.close();
  f.run(1000);
  assert.equal(f.sockets.length, 2);
  first.message({ type: "init", player_id: "stale" });
  assert.equal(f.inits.length, 0);
  f.sockets[1].close();
  f.run(1500);
  assert.equal(f.sockets.length, 3);
  f.session.close();
});

test("leaving cancels watchdog/retry and prevents future sends or stale updates", () => {
  const f = fixture(),
    s = f.sockets[0];
  s.open();
  s.close();
  const callback = s.onmessage;
  f.session.close();
  assert.equal(f.tasks.size, 0);
  assert.equal(f.session.send({ type: "restart" }), false);
  callback({ data: '{"type":"init","player_id":"late"}' });
  assert.equal(f.inits.length, 0);
});

test("watchdog closes a silent connection and schedules reconnection", () => {
  const f = fixture();
  f.sockets[0].open();
  f.run(15000);
  assert.equal(f.statuses.at(-1), "reconnecting");
  f.run(1000);
  assert.equal(f.sockets.length, 2);
  f.session.close();
});

test("input heartbeat resends held keys after server-side respawn resets input", () => {
  const f = fixture(),
    s = f.sockets[0];
  s.open();
  f.run(100);
  assert.equal(s.sent.at(-1).left, true);
  assert.equal(s.sent.at(-1).type, "input");
  f.session.close();
  assert.equal(f.tasks.size, 0);
});

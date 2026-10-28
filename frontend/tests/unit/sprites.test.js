import test from "node:test";
import assert from "node:assert/strict";
import {
  ballPalette,
  hexToRgb,
  tileHash,
  PALETTE,
} from "../../src/renderer/sprites.js";

test("ball palette accepts hex colors and safely falls back for invalid values", () => {
  assert.deepEqual(hexToRgb("#12aBff"), { r: 18, g: 171, b: 255 });
  assert.equal(hexToRgb("red"), null);
  assert.equal(ballPalette("invalid").base, PALETTE.ballMid);
  assert.equal(ballPalette("#ffffff").light, "rgb(255, 255, 255)");
  assert.equal(ballPalette("#123456").trail, "18,52,86");
});

test("tree placement is deterministic for the same tile", () => {
  assert.equal(tileHash(100, 200), tileHash(100, 200));
  assert.notEqual(tileHash(100, 200), tileHash(200, 100));
});

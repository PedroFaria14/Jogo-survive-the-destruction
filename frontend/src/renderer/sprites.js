// Primitivas visuais do Canvas, sem React, rede ou regras de física.
export const PALETTE = {
  skyTop: "#dcece9",
  skyMid: "#f6e3c5",
  skyBottom: "#efc58b",
  grassTop: "#8a9a4a",
  grassEdge: "#6f7d3a",
  soilTop: "#b08d5f",
  soilBottom: "#8f6f47",
  soilBorder: "#6b5a3f",
  islandShadow: "rgba(90,70,40,0.18)",
  falling: "#c9743d",
  fallingDark: "#a8552f",
  treeTrunk: "#7c5a3a",
  treeCanopy: "#6f7d3f",
  treeCanopyLight: "#87944f",
  ballHi: "#ffe3a3",
  ballMid: "#f2b544",
  ballDark: "#d98f24",
  ballGlow: "#f4b942",
  ballMarker: "#7a4b12",
  ballOutline: "rgba(122,75,18,0.6)",
  trailRGB: "244,185,66",
  ghostFill: "rgba(180,150,90,0.18)",
  ghostStroke: "rgba(160,120,60,0.4)",
  gold: "#f4b942",
  amber: "#e0a63c",
  terracotta: "#c96f4a",
  soilCrumb: "#8f6f47",
  cream: "#fdf6e9",
  creamOverlay: "rgba(252,244,231,0.94)",
  textBrown: "#4a3b28",
  textSoft: "#7a623f",
  olive: "#6f7d3a",
  goldStrong: "#d99a2b",
  terracottaStrong: "#c96f4a",
  borderSoft: "rgba(145,110,64,0.55)",
  redMushTop: "#ff6b5e",
  redMushDark: "#c93a3a",
  redMushSpot: "#ffe9dc",
  purpleMushTop: "#b06fd6",
  purpleMushDark: "#7d44a3",
  purpleMushSpot: "#f3e2ff",
  crystalLight: "#cfefff",
  crystalMid: "#5fb9e8",
  crystalDark: "#2f7fb5",
  buffTintRed: "rgba(255,107,94,0.22)",
  buffTintPurple: "rgba(176,111,214,0.22)",
  buffTintBlue: "rgba(95,185,232,0.22)",
};

export function roundRectPath(ctx, x, y, w, h, r) {
  const rr = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.lineTo(x + w - rr, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + rr);
  ctx.lineTo(x + w, y + h - rr);
  ctx.quadraticCurveTo(x + w, y + h, x + w - rr, y + h);
  ctx.lineTo(x + rr, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - rr);
  ctx.lineTo(x, y + rr);
  ctx.quadraticCurveTo(x, y, x + rr, y);
  ctx.closePath();
}

export function hexToRgb(hex) {
  const m = /^#([0-9a-fA-F]{6})$/.exec(hex || "");
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
}

function shadeHex(hex, factor) {
  const c = hexToRgb(hex);
  const to = (v) => Math.round(Math.min(255, Math.max(0, v * factor)));
  return `rgb(${to(c.r)}, ${to(c.g)}, ${to(c.b)})`;
}

export function ballPalette(hex) {
  const c = hexToRgb(hex);
  if (!c)
    return {
      base: PALETTE.ballMid,
      light: PALETTE.ballHi,
      dark: PALETTE.ballDark,
      marker: PALETTE.ballMarker,
      outline: PALETTE.ballOutline,
      trail: PALETTE.trailRGB,
      glow: PALETTE.ballGlow,
    };
  return {
    base: hex,
    light: shadeHex(hex, 1.5),
    dark: shadeHex(hex, 0.6),
    marker: shadeHex(hex, 0.35),
    outline: `rgba(${c.r}, ${c.g}, ${c.b}, 0.6)`,
    trail: `${c.r},${c.g},${c.b}`,
    glow: hex,
  };
}

export function tileHash(x, y) {
  let h = 17;
  h = (h * 31 + Math.round(x)) | 0;
  h = (h * 31 + Math.round(y)) | 0;
  return Math.abs(h);
}

export function drawTree(ctx, x, y, s) {
  ctx.fillStyle = "rgba(60,70,30,0.15)";
  ctx.beginPath();
  ctx.ellipse(x + s * 0.5, y + s * 0.62, s * 0.18, s * 0.06, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = PALETTE.treeTrunk;
  roundRectPath(ctx, x + s * 0.46, y + s * 0.46, s * 0.08, s * 0.18, 3);
  ctx.fill();
  ctx.fillStyle = PALETTE.treeCanopy;
  ctx.beginPath();
  ctx.arc(x + s * 0.5, y + s * 0.34, s * 0.22, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = PALETTE.treeCanopyLight;
  ctx.beginPath();
  ctx.arc(x + s * 0.43, y + s * 0.27, s * 0.12, 0, Math.PI * 2);
  ctx.fill();
}

export function drawCloud(ctx, x, y, s) {
  ctx.fillStyle = "rgba(255,255,255,0.5)";
  ctx.beginPath();
  ctx.ellipse(x, y, s, s * 0.4, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(x - s * 0.6, y + s * 0.15, s * 0.55, s * 0.32, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(x + s * 0.6, y + s * 0.12, s * 0.5, s * 0.3, 0, 0, Math.PI * 2);
  ctx.fill();
}

function mushroomCap(ctx, x, y) {
  ctx.beginPath();
  ctx.moveTo(x - 13, y + 2);
  ctx.quadraticCurveTo(x - 13, y - 13, x, y - 13);
  ctx.quadraticCurveTo(x + 13, y - 13, x + 13, y + 2);
}

function drawMushroom(ctx, x, y, type) {
  const red = type === "red_mushroom";
  ctx.fillStyle = "#f6ead2";
  roundRectPath(ctx, x - 4, y + 2, 8, 9, 3);
  ctx.fill();
  ctx.fillStyle = red ? PALETTE.redMushTop : PALETTE.purpleMushTop;
  mushroomCap(ctx, x, y);
  ctx.closePath();
  ctx.fill();
  ctx.fillStyle = "rgba(255,255,255,0.35)";
  ctx.beginPath();
  ctx.ellipse(x - 5, y - 7, 4, 2.4, -0.5, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = red ? PALETTE.redMushSpot : PALETTE.purpleMushSpot;
  ctx.beginPath();
  ctx.arc(x - 6, y - 4, 1.8, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(x + 6, y - 2, 1.4, 0, Math.PI * 2);
  ctx.fill();
  ctx.strokeStyle = red ? PALETTE.redMushDark : PALETTE.purpleMushDark;
  ctx.lineWidth = 1.5;
  mushroomCap(ctx, x, y);
  ctx.stroke();
}

function drawCrystal(ctx, x, y) {
  ctx.fillStyle = PALETTE.crystalDark;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x + 9, y - 4);
  ctx.lineTo(x + 6, y + 9);
  ctx.lineTo(x - 6, y + 9);
  ctx.lineTo(x - 9, y - 4);
  ctx.closePath();
  ctx.fill();
  ctx.fillStyle = PALETTE.crystalMid;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x + 9, y - 4);
  ctx.lineTo(x, y);
  ctx.closePath();
  ctx.fill();
  ctx.fillStyle = PALETTE.crystalLight;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x - 9, y - 4);
  ctx.lineTo(x - 3, y - 4);
  ctx.closePath();
  ctx.fill();
  ctx.strokeStyle = PALETTE.crystalDark;
  ctx.lineWidth = 1.2;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x, y);
  ctx.moveTo(x - 9, y - 4);
  ctx.lineTo(x + 9, y - 4);
  ctx.stroke();
}

export function drawPowerUpDrop(ctx, pu, now) {
  const bob = Math.sin(now / 320 + (pu.id ? pu.id.length : 0)) * 2;
  const glow =
    pu.type === "red_mushroom"
      ? PALETTE.redMushTop
      : pu.type === "purple_mushroom"
        ? PALETTE.purpleMushTop
        : PALETTE.crystalMid;
  ctx.save();
  ctx.translate(pu.x, pu.y + bob);
  ctx.shadowColor = glow;
  ctx.shadowBlur = 14;
  ctx.fillStyle = glow;
  ctx.globalAlpha = 0.16 + Math.sin(now / 260) * 0.05;
  ctx.beginPath();
  ctx.arc(0, 0, 15, 0, Math.PI * 2);
  ctx.fill();
  ctx.globalAlpha = 1;
  if (pu.type === "blue_crystal") drawCrystal(ctx, 0, 0);
  else drawMushroom(ctx, 0, 0, pu.type);
  ctx.restore();
}

export function drawPlayerBall(ctx, player, anim, now, dt, radius, buff) {
  const pal = ballPalette(player.color);
  const buffHex =
    buff === "red_mushroom"
      ? PALETTE.redMushTop
      : buff === "purple_mushroom"
        ? PALETTE.purpleMushTop
        : buff === "blue_crystal"
          ? PALETTE.crystalMid
          : null;
  const rgb = hexToRgb(buffHex);
  const trailColor = rgb ? `${rgb.r},${rgb.g},${rgb.b}` : pal.trail;
  const glowColor =
    buff === "red_mushroom"
      ? PALETTE.redMushDark
      : buff === "purple_mushroom"
        ? PALETTE.purpleMushDark
        : buff === "blue_crystal"
          ? PALETTE.crystalLight
          : pal.glow;
  anim.trail.push({ x: player.x, y: player.y });
  if (anim.trail.length > 12) anim.trail.shift();
  anim.trail.forEach((point, i) => {
    ctx.fillStyle = `rgba(${trailColor},${((i / 12) * 0.22).toFixed(3)})`;
    ctx.beginPath();
    ctx.arc(point.x, point.y, radius * 0.55 * (i / 12), 0, Math.PI * 2);
    ctx.fill();
  });
  if (Math.abs(player.vx) > 0.1) anim.rotation += (player.vx * dt) / radius;
  let scaleX = 1,
    scaleY = 1;
  if (player.on_ground) {
    scaleY = 1 - anim.landSquash * 0.35;
    scaleX = 1 + anim.landSquash * 0.35;
    anim.landSquash = Math.max(0, anim.landSquash - dt * 2.5);
  } else {
    const stretch = Math.min(Math.abs(player.vy) / 420, 0.32);
    scaleY = 1 + stretch;
    scaleX = 1 - stretch * 0.6;
  }
  if (player.on_ground && !anim.prevOnGround && anim.prevVy > 8)
    anim.landSquash = Math.min((anim.prevVy - 8) / 28, 1);
  anim.prevOnGround = player.on_ground;
  anim.prevVy = player.vy;
  ctx.save();
  ctx.translate(player.x, player.y);
  ctx.scale(scaleX, scaleY);
  ctx.shadowColor = glowColor;
  ctx.shadowBlur = 16 + Math.sin(now / 260) * 6;
  if (buff) {
    ctx.fillStyle =
      buff === "red_mushroom"
        ? PALETTE.buffTintRed
        : buff === "purple_mushroom"
          ? PALETTE.buffTintPurple
          : PALETTE.buffTintBlue;
    ctx.beginPath();
    ctx.arc(0, 0, radius, 0, Math.PI * 2);
    ctx.fill();
  }
  const grad = ctx.createRadialGradient(-6, -6, 4, 0, 0, radius);
  grad.addColorStop(0, pal.light);
  grad.addColorStop(0.55, pal.base);
  grad.addColorStop(1, pal.dark);
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.shadowBlur = 0;
  ctx.save();
  ctx.rotate(anim.rotation);
  ctx.fillStyle = pal.marker;
  ctx.beginPath();
  ctx.arc(radius * 0.55, 0, 4.5, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
  ctx.strokeStyle = pal.outline;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.stroke();
  if (buff === "blue_crystal") {
    ctx.strokeStyle = "rgba(207,239,255,0.7)";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    for (let a = 0; a < Math.PI * 2; a += 0.3) {
      const rx = radius * 1.35 * Math.cos(a),
        ry = radius * 1.35 * Math.sin(a) * 0.5;
      ctx.moveTo(rx * 0.85, ry * 0.85);
      ctx.lineTo(rx, ry);
    }
    ctx.stroke();
  }
  ctx.restore();
}

export function drawGhost(ctx, player, radius) {
  const c = hexToRgb(player.color);
  ctx.fillStyle = c ? `rgba(${c.r}, ${c.g}, ${c.b}, 0.18)` : PALETTE.ghostFill;
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.strokeStyle = c
    ? `rgba(${c.r}, ${c.g}, ${c.b}, 0.4)`
    : PALETTE.ghostStroke;
  ctx.lineWidth = 1.5;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.setLineDash([]);
}

export function drawBuffIcon(ctx, x, y, buff, now) {
  ctx.save();
  ctx.translate(x, y - 34 + Math.sin(now / 280) * 2);
  if (buff === "blue_crystal") drawCrystal(ctx, 0, 0);
  else drawMushroom(ctx, 0, 0, buff);
  ctx.restore();
}

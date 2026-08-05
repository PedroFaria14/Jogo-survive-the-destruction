// Predição client-side: réplica da física do jogador local para eliminar o
// delay de resposta ao input (latência de rede). Espelha o backend em
// backend/models/movement.go (ApplyPhysics/ProcessInput) e powerup.go.
// O servidor continua autoritativo: a cada snapshot recebido, o sim é
// reconciliado (correção suave ou snap) com a posição oficial.

export const PHYS = {
  MoveSpeed: 8,
  JumpForce: 20,
  Gravity: 1.5,
  VariableJumpFactor: 0.6,
  Friction: 0.85,
  AirResistance: 0.98,
  CoyoteTimeMs: 120,
  JumpBufferMs: 120,
  MaxAirJumps: 1,
  DashSpeed: 20,
  DashCooldownMs: 1500,
  DashDurationMs: 250,
  MaxSpeed: 34,
  TankRadiusScale: 2.0,
  TankMass: 2.0,
  SpeedMult: 1.8,
  GlideGravity: 0.2,
  KnockbackBase: 2.0,
  KnockbackDash: 11.0,
  DashRecoil: 0.3,
  Restitution: 0.85,
};

const TILE_MS = 16.666; // ~60 FPS, igual ao tick do Hub

// Cria o estado do sim a partir de um snapshot do servidor.
export function createSim(player) {
  return {
    x: player.x,
    y: player.y,
    vx: player.vx || 0,
    vy: player.vy || 0,
    on_ground: !!player.on_ground,
    jumps_used: player.jumps_used || 0,
    jump_held: false,
    jump_buffered_at: null,
    last_grounded_at: null,
    left_held: false,
    right_held: false,
    dash_held: false,
    dash_queued: false,
    dash_until: 0,
    dash_ready_at: 0,
    buff: player.buff || '',
    buff_until: 0,
    is_dead: !!player.is_dead,
    player_radius: player.player_radius,
  };
}

// ProcessInput: aplica o estado das teclas no sim (edge-detect do dash,
// jump buffer e pulo variável). Deve rodar no momento do input.
export function applyInput(sim, input, nowMs) {
  if (sim.is_dead) return;
  sim.left_held = !!input.left;
  sim.right_held = !!input.right;

  if (input.jump) {
    sim.jump_buffered_at = nowMs;
  }
  sim.jump_held = !!input.jump;

  if (input.dash && !sim.dash_held) {
    if (nowMs >= sim.dash_ready_at) {
      sim.dash_queued = true;
    }
  }
  sim.dash_held = !!input.dash;
}

function buffActive(sim, nowMs) {
  return sim.buff !== '' && nowMs < sim.buff_until;
}

function radiusOf(sim, nowMs, baseRadius) {
  return sim.buff === 'red_mushroom' && buffActive(sim, nowMs)
    ? baseRadius * PHYS.TankRadiusScale
    : baseRadius;
}

function speedMult(sim, nowMs) {
  return sim.buff === 'purple_mushroom' && buffActive(sim, nowMs)
    ? PHYS.SpeedMult
    : 1;
}

function gravityScale(sim, nowMs) {
  return sim.buff === 'blue_crystal' && buffActive(sim, nowMs)
    ? PHYS.GlideGravity
    : 1;
}

// Um passo de física (equivalente a um tick do servidor).
export function stepPhysics(sim, tiles, nowMs, cfg) {
  if (sim.is_dead) return;

  const radius = radiusOf(sim, nowMs, cfg.player_radius ?? 25);
  const speed = (cfg.move_speed ?? PHYS.MoveSpeed) * speedMult(sim, nowMs);

  // Dash agendado (edge-detect feito no applyInput).
  if (sim.dash_queued) {
    let facing = 1;
    if (sim.left_held) facing = -1;
    else if (sim.right_held) facing = 1;
    else if (sim.vx !== 0) facing = Math.sign(sim.vx);
    sim.vx = facing * (cfg.dash_speed ?? PHYS.DashSpeed);
    sim.dash_until = nowMs + (cfg.dash_duration ?? PHYS.DashDurationMs / 1000) * 1000;
    sim.dash_ready_at = nowMs + (cfg.dash_cooldown ?? PHYS.DashCooldownMs / 1000) * 1000;
    sim.dash_queued = false;
  }

  const dashing = nowMs < sim.dash_until;

  // Pulo: jump buffer + coyote time + pulo duplo.
  if (sim.jump_buffered_at !== null && nowMs - sim.jump_buffered_at <= PHYS.JumpBufferMs) {
    const onGround =
      sim.on_ground ||
      (sim.last_grounded_at !== null && nowMs - sim.last_grounded_at <= PHYS.CoyoteTimeMs);
    if (onGround) {
      sim.vy = -(cfg.jump_force ?? PHYS.JumpForce);
      sim.on_ground = false;
      sim.last_grounded_at = null;
      sim.jump_buffered_at = null;
      sim.jumps_used = 0;
    } else if (sim.jumps_used < (cfg.max_air_jumps ?? PHYS.MaxAirJumps)) {
      sim.vy = -(cfg.jump_force ?? PHYS.JumpForce);
      sim.jumps_used++;
      sim.jump_buffered_at = null;
    }
  }

  // Forças e resistência.
  if (dashing) {
    // Mantém a velocidade do dash sem recontrole.
  } else if (sim.left_held) {
    sim.vx = -speed;
  } else if (sim.right_held) {
    sim.vx = speed;
  } else if (sim.on_ground) {
    sim.vx *= PHYS.Friction;
  }
  if (!sim.on_ground && !dashing) {
    sim.vx *= PHYS.AirResistance;
  }

  // Movimento horizontal + colisão lateral com tiles.
  sim.x += sim.vx;
  for (const tile of tiles) {
    if (!tile.is_active) continue;
    if (sim.y + radius > tile.y && sim.y - radius < tile.y + cfg.tile_size) {
      if (sim.vx > 0 && sim.x + radius > tile.x && sim.x - radius < tile.x) {
        sim.x = tile.x - radius;
        sim.vx = 0;
      }
      if (sim.vx < 0 && sim.x - radius < tile.x + cfg.tile_size && sim.x + radius > tile.x + cfg.tile_size) {
        sim.x = tile.x + cfg.tile_size + radius;
        sim.vx = 0;
      }
    }
  }

  // Limites da arena.
  if (sim.x <= radius) {
    sim.x = radius;
    sim.vx = 0;
  }
  if (sim.x >= (cfg.arena_width ?? 800) - radius) {
    sim.x = (cfg.arena_width ?? 800) - radius;
    sim.vx = 0;
  }

  // Movimento vertical: gravidade (pulo variável reduz ao subir).
  let g = (cfg.gravity ?? PHYS.Gravity) * gravityScale(sim, nowMs);
  if (sim.jump_held && sim.vy < 0) {
    g = (cfg.gravity ?? PHYS.Gravity) * PHYS.VariableJumpFactor * gravityScale(sim, nowMs);
  }
  sim.vy += g;
  sim.y += sim.vy;

  // Colisão com teto (subindo).
  if (sim.vy < 0) {
    for (const tile of tiles) {
      if (
        tile.is_active &&
        sim.x > tile.x &&
        sim.x < tile.x + cfg.tile_size &&
        sim.y - radius < tile.y + cfg.tile_size &&
        sim.y - radius > tile.y
      ) {
        sim.y = tile.y + cfg.tile_size + radius;
        sim.vy = 0;
        break;
      }
    }
  }

  // Colisão de pouso.
  let hit = false;
  for (const tile of tiles) {
    if (
      tile.is_active &&
      sim.vy > 0 &&
      sim.x > tile.x &&
      sim.x < tile.x + cfg.tile_size &&
      sim.y + radius > tile.y &&
      sim.y + radius < tile.y + cfg.tile_size
    ) {
      sim.y = tile.y - radius;
      sim.vy = 0;
      sim.on_ground = true;
      sim.last_grounded_at = nowMs;
      sim.jumps_used = 0;
      hit = true;
      break;
    }
  }
  if (!hit) sim.on_ground = false;
}

// Reconcilia o sim com o snapshot autoritativo do servidor.
export function reconcileSim(sim, player, cfg, nowMs) {
  const err = Math.hypot(player.x - sim.x, player.y - sim.y);
  const snapThreshold = cfg.snap_threshold ?? 60;
  const blendThreshold = cfg.blend_threshold ?? 40;

  // Campos autoritativos sempre sincronizados.
  sim.buff = player.buff || '';
  sim.buff_until = sim.buff ? nowMs + (player.buff_remaining ?? 0) * 1000 : 0;
  sim.on_ground = !!player.on_ground;
  sim.jumps_used = player.jumps_used || 0;
  if (sim.on_ground) sim.last_grounded_at = nowMs;
  sim.dash_ready_at = nowMs + (player.dash_cd ?? 0) * 1000;

  if (player.is_dead) {
    sim.is_dead = true;
    return;
  }
  if (sim.is_dead) {
    // Ressuscitou (respawn): sincroniza completamente.
    sim.is_dead = false;
    snapTo(sim, player);
    return;
  }

  if (err > snapThreshold) {
    snapTo(sim, player);
  } else if (err > blendThreshold) {
    // Erro grande (ex.: knockback de outro jogador): corrige rápido.
    const k = 0.35;
    sim.x += (player.x - sim.x) * k;
    sim.y += (player.y - sim.y) * k;
    copyVelocity(sim, player);
  } else {
    // Erro pequeno: mantém a predição, mas adota a velocidade oficial para
    // propagar knockback/buffs sem teleporte.
    copyVelocity(sim, player);
  }
}

function snapTo(sim, player) {
  sim.x = player.x;
  sim.y = player.y;
  sim.vx = player.vx || 0;
  sim.vy = player.vy || 0;
}

function copyVelocity(sim, player) {
  sim.vx = player.vx ?? sim.vx;
  sim.vy = player.vy ?? sim.vy;
}

// Avança o sim em passos fixos equivalentes ao tick do servidor (~60 FPS).
export function stepSim(sim, tiles, nowMs, cfg, maxSteps) {
  let remaining = maxSteps;
  while (remaining-- > 0) {
    stepPhysics(sim, tiles, nowMs, cfg);
  }
}

// Quantos passos de 16ms cabem no dt decorrido (com teto anti-espiral).
export function ticksFor(dtMs) {
  const ticks = Math.floor(dtMs / TILE_MS);
  return Math.max(1, Math.min(ticks, 5));
}

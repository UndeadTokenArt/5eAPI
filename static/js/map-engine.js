// Edge opposite direction
const OPP = { N:'S', S:'N', E:'W', W:'E' };
const DIRS = ['N','E','S','W'];
// Neighbour offsets [dx, dy] per direction
const OFFSET = { N:[0,-1], E:[1,0], S:[0,1], W:[-1,0] };
const TILE_SVG_CACHE = new Map();

// ═══════════════════════════════════════════════════════════════════════════════
// WAVE FUNCTION COLLAPSE
// ═══════════════════════════════════════════════════════════════════════════════

function wfcGenerate(mapType) {
  const typeTiles = TILES.filter(t => t.type === mapType);

  // Each cell: Set of candidate tile IDs
  function initGrid() {
    const g = [];
    for (let y = 0; y < GRID; y++) {
      g[y] = [];
      for (let x = 0; x < GRID; x++) {
        g[y][x] = new Set(typeTiles.map(t => t.id));
      }
    }
    return g;
  }

  function collapsedNeighborIds(grid, x, y) {
    const ids = [];
    for (const dir of DIRS) {
      const [dx, dy] = OFFSET[dir];
      const nx = x + dx, ny = y + dy;
      if (nx < 0 || nx >= GRID || ny < 0 || ny >= GRID) continue;
      const neighbor = grid[ny][nx];
      if (neighbor.size !== 1) continue;
      ids.push(Array.from(neighbor)[0]);
    }
    return ids;
  }

  function ruleMatchesNeighbor(rule, neighborId) {
    if (!rule || !neighborId) return false;

    const tileMatch = Array.isArray(rule.tileIds) && rule.tileIds.includes(neighborId);

    let groupMatch = false;
    if (Array.isArray(rule.groups) && rule.groups.length > 0) {
      const neighborGroups = TILE_GROUPS_BY_ID[neighborId] || [];
      groupMatch = rule.groups.some(group => neighborGroups.includes(group));
    }

    return tileMatch || groupMatch;
  }

  function effectiveWeightFor(tileId, neighborIds) {
    const tile = TILE_BY_ID[tileId];
    if (!tile) return 0;

    const baseWeight = Number(tile.weight) || 0;
    const rules = Array.isArray(tile.neighborWeightRules) ? tile.neighborWeightRules : [];
    if (rules.length === 0 || neighborIds.length === 0) return Math.max(0, baseWeight);

    let bestOverride = null;
    let bestStrength = -1;

    for (const neighborId of neighborIds) {
      for (const rule of rules) {
        if (!ruleMatchesNeighbor(rule, neighborId)) continue;

        const override = Number(rule.weightOverride);
        if (!Number.isFinite(override)) continue;

        const strength = Math.abs(override - baseWeight);
        if (
          strength > bestStrength ||
          (strength === bestStrength && (bestOverride === null || override < bestOverride))
        ) {
          bestStrength = strength;
          bestOverride = override;
        }
      }
    }

    const effective = bestOverride === null ? baseWeight : bestOverride;
    if (!Number.isFinite(effective) || effective < 0) return 0;
    return effective;
  }

  // Weighted random pick from an array of tile IDs with neighbor-aware overrides
  function weightedPick(ids, grid, x, y) {
    const neighborIds = collapsedNeighborIds(grid, x, y);
    const candidates = ids.map(id => ({ id, weight: effectiveWeightFor(id, neighborIds) }));

    let total = 0;
    candidates.forEach(c => { total += c.weight; });
    if (total <= 0) return null;

    let r = Math.random() * total;
    for (const candidate of candidates) {
      if (candidate.weight <= 0) continue;
      r -= candidate.weight;
      if (r <= 0) return candidate.id;
    }

    for (const candidate of candidates) {
      if (candidate.weight > 0) return candidate.id;
    }
    return null;
  }

  // Find the uncollapsed cell with minimum entropy (fewest options)
  function minEntropyCell(grid) {
    let best = null, bestN = Infinity;
    for (let y = 0; y < GRID; y++) {
      for (let x = 0; x < GRID; x++) {
        const n = grid[y][x].size;
        if (n > 1 && n < bestN) { bestN = n; best = {x, y}; }
      }
    }
    return best;
  }

  // Propagate constraints after collapsing cell (x,y)
  function propagate(grid, startX, startY) {
    const queue = [{x: startX, y: startY}];
    while (queue.length > 0) {
      const {x, y} = queue.shift();
      const cellOptions = grid[y][x];

      for (const dir of DIRS) {
        const [dx, dy] = OFFSET[dir];
        const nx = x + dx, ny = y + dy;
        if (nx < 0 || nx >= GRID || ny < 0 || ny >= GRID) continue;

        const neighbor = grid[ny][nx];

        // Collect edge values that cell (x,y) can produce toward dir
        const allowedEdges = new Set();
        for (const id of cellOptions) {
          allowedEdges.add(TILE_BY_ID[id].edges[dir]);
        }

        // Remove from neighbor any tile whose opposite-dir edge is not in allowedEdges
        const opp = OPP[dir];
        let changed = false;
        for (const id of Array.from(neighbor)) {
          if (!allowedEdges.has(TILE_BY_ID[id].edges[opp])) {
            neighbor.delete(id);
            changed = true;
          }
        }
        if (neighbor.size === 0) return false; // contradiction
        if (changed) queue.push({x: nx, y: ny});
      }
    }
    return true;
  }

  // Main WFC loop
  function attempt() {
    const grid = initGrid();

    while (true) {
      const cell = minEntropyCell(grid);
      if (!cell) break; // all collapsed

      const {x, y} = cell;
      const chosen = weightedPick(Array.from(grid[y][x]), grid, x, y);
      if (!chosen) return null; // all effective weights are zero in this cell
      grid[y][x] = new Set([chosen]);

      if (!propagate(grid, x, y)) return null; // contradiction
    }
    return grid;
  }

  // Retry up to 8 times on contradiction
  for (let i = 0; i < 8; i++) {
    const grid = attempt();
    if (grid) return grid;
  }
  return null;
}

// ═══════════════════════════════════════════════════════════════════════════════
// SVG RENDERING
// ═══════════════════════════════════════════════════════════════════════════════

function mkSVGGroup(id, content) {
  return '<g id="tile-' + id + '">' + content + '</g>';
}

async function getTileSVG(tile) {
  const key = tile.type + '/' + tile.asset;

  const overrideMap = (typeof window !== 'undefined' && window.ACTIVE_SVG_BY_ASSET)
    ? window.ACTIVE_SVG_BY_ASSET
    : null;
  if (overrideMap && typeof overrideMap[key] === 'string' && overrideMap[key].trim()) {
    const overrideRaw = overrideMap[key];
    const overrideInner = overrideRaw
      .replace(/^[\s\S]*?<svg[^>]*>/i, '')
      .replace(/<\/svg>[\s\S]*$/i, '')
      .trim();
    return overrideInner || overrideRaw;
  }

  if (TILE_SVG_CACHE.has(key)) {
    return TILE_SVG_CACHE.get(key);
  }

  const res = await fetch('/tiles/' + key + '.svg', { cache: 'no-store' });
  if (!res.ok) {
    throw new Error('Failed to load tile SVG: ' + key + ' (' + res.status + ')');
  }

  const raw = await res.text();
  const inner = raw
    .replace(/^[\s\S]*?<svg[^>]*>/i, '')
    .replace(/<\/svg>[\s\S]*$/i, '')
    .trim();

  TILE_SVG_CACHE.set(key, inner);
  return inner;
}

async function buildTileDefs() {
  const defs = document.getElementById('tile-defs');
  const groups = await Promise.all(
    TILES.map(async tile => mkSVGGroup(tile.id, await getTileSVG(tile)))
  );
  defs.innerHTML = groups.join('\n');
}

function renderGrid(grid) {
  const layer = document.getElementById('map-layer');
  const fragments = [];
  for (let y = 0; y < GRID; y++) {
    for (let x = 0; x < GRID; x++) {
      const ids = Array.from(grid[y][x]);
      const id = ids.length > 0 ? ids[0] : '';
      if (!id) continue;
      const tx = x * TILE;
      const ty = y * TILE;
      fragments.push(
        '<use href="#tile-' + id + '" transform="translate(' + tx + ',' + ty + ')"/>'
      );
    }
  }
  layer.innerHTML = fragments.join('');
}

// ═══════════════════════════════════════════════════════════════════════════════
// STATE
// ═══════════════════════════════════════════════════════════════════════════════
let currentMapType = 'dungeon';
let currentTheme   = THEMES[0];
let currentZoom    = 1.0;

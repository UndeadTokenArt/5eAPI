// Edge opposite direction
const OPP = { N:'S', S:'N', E:'W', W:'E' };
const DIRS = ['N','E','S','W'];
// Neighbour offsets [dx, dy] per direction
const OFFSET = { N:[0,-1], E:[1,0], S:[0,1], W:[-1,0] };
const TILE_SVG_CACHE = new Map();
const ROOM_LAYOUT_TYPES = new Set(['dungeon', 'indoor', 'castle']);
const DEFAULT_ROOM_COUNT = 5;
const DEFAULT_ROOM_SIZE_PRESET = 'varied';
const ROOM_SIZE_PRESETS = {
  compact: { min: 1, max: 2 },
  varied: { min: 1, max: 3 },
  broad: { min: 2, max: 4 },
};

function inBounds(x, y) {
  return x >= 0 && x < GRID && y >= 0 && y < GRID;
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function roomPlanningSettings() {
  const shared = (typeof window !== 'undefined' && window.MAP_ROOM_SETTINGS)
    ? window.MAP_ROOM_SETTINGS
    : {};

  const rawCount = Number(shared.count);
  const count = Number.isFinite(rawCount) ? clamp(Math.round(rawCount), 1, 8) : DEFAULT_ROOM_COUNT;
  const sizePreset = ROOM_SIZE_PRESETS[shared.sizePreset] ? shared.sizePreset : DEFAULT_ROOM_SIZE_PRESET;
  return { count, sizePreset, sizeRange: ROOM_SIZE_PRESETS[sizePreset] };
}

function initBiasGrid() {
  const grid = [];
  for (let y = 0; y < GRID; y++) {
    grid[y] = [];
    for (let x = 0; x < GRID; x++) {
      grid[y][x] = Object.create(null);
    }
  }
  return grid;
}

function addBias(cellBiases, x, y, tileId, multiplier) {
  if (!inBounds(x, y) || !tileId || !Number.isFinite(multiplier) || multiplier <= 0) return;
  const cell = cellBiases[y][x];
  cell[tileId] = Math.max(cell[tileId] || 1, multiplier);
}

function addBiasSet(cellBiases, x, y, biasSet) {
  Object.entries(biasSet).forEach(([tileId, multiplier]) => {
    addBias(cellBiases, x, y, tileId, multiplier);
  });
}

function chooseRoomLayouts(settings) {
  const rooms = [];
  const maxRadius = settings.sizeRange.max;
  const margin = maxRadius + 2;
  let attempts = settings.count * 40;

  while (rooms.length < settings.count && attempts > 0) {
    attempts -= 1;
    const x = margin + Math.floor(Math.random() * Math.max(1, GRID - (margin * 2)));
    const y = margin + Math.floor(Math.random() * Math.max(1, GRID - (margin * 2)));
    const radius = settings.sizeRange.min + Math.floor(Math.random() * (settings.sizeRange.max - settings.sizeRange.min + 1));

    const tooClose = rooms.some(room => {
      const dx = room.x - x;
      const dy = room.y - y;
      const minSpacing = room.radius + radius + 3;
      return Math.abs(dx) + Math.abs(dy) < minSpacing;
    });
    if (tooClose) continue;

    rooms.push({ x, y, radius });
  }

  if (rooms.length === 0) {
    rooms.push({ x: Math.floor(GRID / 2), y: Math.floor(GRID / 2), radius: settings.sizeRange.min });
  }

  return rooms;
}

function roomInteriorBiases() {
  return {
    'd-floor': 14,
    'd-pillar': 2.5,
    'd-brazier': 1.8,
    'd-torch': 1.6,
    'd-chest': 1.3,
    'd-altar': 1.2,
  };
}

function roomEdgeBiases(edge) {
  const edgeMap = {
    N: { 'd-t-n': 55, 'd-door-v': 8, 'd-floor': 4 },
    E: { 'd-t-e': 55, 'd-door-h': 8, 'd-floor': 4 },
    S: { 'd-t-s': 55, 'd-door-v': 8, 'd-floor': 4 },
    W: { 'd-t-w': 55, 'd-door-h': 8, 'd-floor': 4 },
  };
  return edgeMap[edge] || { 'd-floor': 3 };
}

function roomCornerBiases(verticalEdge, horizontalEdge) {
  const key = verticalEdge + horizontalEdge;
  const cornerMap = {
    NW: { 'd-cor-se': 90, 'd-floor': 3 },
    NE: { 'd-cor-sw': 90, 'd-floor': 3 },
    SW: { 'd-cor-ne': 90, 'd-floor': 3 },
    SE: { 'd-cor-nw': 90, 'd-floor': 3 },
  };
  return cornerMap[key] || { 'd-floor': 3 };
}

function paintRoomBiases(cellBiases, room) {
  for (let dy = -room.radius; dy <= room.radius; dy++) {
    for (let dx = -room.radius; dx <= room.radius; dx++) {
      const x = room.x + dx;
      const y = room.y + dy;
      if (!inBounds(x, y)) continue;

      const atNorth = dy === -room.radius;
      const atSouth = dy === room.radius;
      const atWest = dx === -room.radius;
      const atEast = dx === room.radius;

      if ((atNorth || atSouth) && (atWest || atEast)) {
        const verticalEdge = atNorth ? 'N' : 'S';
        const horizontalEdge = atWest ? 'W' : 'E';
        addBiasSet(cellBiases, x, y, roomCornerBiases(verticalEdge, horizontalEdge));
        continue;
      }

      if (atNorth) {
        addBiasSet(cellBiases, x, y, roomEdgeBiases('N'));
        continue;
      }
      if (atSouth) {
        addBiasSet(cellBiases, x, y, roomEdgeBiases('S'));
        continue;
      }
      if (atWest) {
        addBiasSet(cellBiases, x, y, roomEdgeBiases('W'));
        continue;
      }
      if (atEast) {
        addBiasSet(cellBiases, x, y, roomEdgeBiases('E'));
        continue;
      }

      addBiasSet(cellBiases, x, y, roomInteriorBiases());
    }
  }

  addBias(cellBiases, room.x, room.y, 'd-floor', 18);
}

function directionBetween(from, to) {
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  if (dx === 1 && dy === 0) return 'E';
  if (dx === -1 && dy === 0) return 'W';
  if (dx === 0 && dy === 1) return 'S';
  if (dx === 0 && dy === -1) return 'N';
  return null;
}

function exitForRoom(room, target) {
  const dx = target.x - room.x;
  const dy = target.y - room.y;

  if (Math.abs(dx) >= Math.abs(dy)) {
    const dir = dx >= 0 ? 'E' : 'W';
    return {
      door: { x: room.x + (dir === 'E' ? room.radius : -room.radius), y: room.y, dir },
      outside: { x: room.x + (dir === 'E' ? room.radius + 1 : -(room.radius + 1)), y: room.y },
    };
  }

  const dir = dy >= 0 ? 'S' : 'N';
  return {
    door: { x: room.x, y: room.y + (dir === 'S' ? room.radius : -room.radius), dir },
    outside: { x: room.x, y: room.y + (dir === 'S' ? room.radius + 1 : -(room.radius + 1)) },
  };
}

function markPathConnection(pathDirs, from, to) {
  if (!inBounds(from.x, from.y) || !inBounds(to.x, to.y)) return;

  const xFirst = Math.random() < 0.5;
  const points = [{ x: from.x, y: from.y }];
  let x = from.x;
  let y = from.y;

  function marchTo(targetX, targetY) {
    while (x !== targetX || y !== targetY) {
      if (x !== targetX) {
        x += x < targetX ? 1 : -1;
      } else if (y !== targetY) {
        y += y < targetY ? 1 : -1;
      }
      points.push({ x, y });
    }
  }

  if (xFirst) {
    marchTo(to.x, from.y);
    marchTo(to.x, to.y);
  } else {
    marchTo(from.x, to.y);
    marchTo(to.x, to.y);
  }

  for (let i = 0; i < points.length - 1; i++) {
    const curr = points[i];
    const next = points[i + 1];
    const dir = directionBetween(curr, next);
    if (!dir) continue;
    pathDirs[curr.y][curr.x].add(dir);
    pathDirs[next.y][next.x].add(OPP[dir]);
  }
}

function initPathDirGrid() {
  const grid = [];
  for (let y = 0; y < GRID; y++) {
    grid[y] = [];
    for (let x = 0; x < GRID; x++) {
      grid[y][x] = new Set();
    }
  }
  return grid;
}

function corridorBiasesForDirs(openDirs) {
  const dirs = Array.from(openDirs).sort().join('');
  const byShape = {
    EW: { 'd-corr-h': 36, 'd-door-h': 4 },
    NS: { 'd-corr-v': 36, 'd-door-v': 4 },
    EN: { 'd-cor-ne': 70, 'd-floor': 2 },
    ES: { 'd-cor-se': 70, 'd-floor': 2 },
    SW: { 'd-cor-sw': 70, 'd-floor': 2 },
    NW: { 'd-cor-nw': 70, 'd-floor': 2 },
    ESW: { 'd-t-n': 65, 'd-floor': 2 },
    NSW: { 'd-t-e': 65, 'd-floor': 2 },
    ENW: { 'd-t-s': 65, 'd-floor': 2 },
    ENS: { 'd-t-w': 65, 'd-floor': 2 },
    ENSW: { 'd-cross': 200, 'd-floor': 2 },
  };
  return byShape[dirs] || { 'd-floor': 2 };
}

function paintCorridorBiases(cellBiases, pathDirs) {
  for (let y = 0; y < GRID; y++) {
    for (let x = 0; x < GRID; x++) {
      const openDirs = pathDirs[y][x];
      if (!openDirs || openDirs.size < 2) continue;
      addBiasSet(cellBiases, x, y, corridorBiasesForDirs(openDirs));
    }
  }
}

function paintDoorBiases(cellBiases, doors) {
  doors.forEach(door => {
    if (!inBounds(door.x, door.y)) return;
    if (door.dir === 'E' || door.dir === 'W') {
      addBiasSet(cellBiases, door.x, door.y, { 'd-door-h': 85, 'd-corr-h': 10, 'd-floor': 2 });
    } else {
      addBiasSet(cellBiases, door.x, door.y, { 'd-door-v': 85, 'd-corr-v': 10, 'd-floor': 2 });
    }
  });
}

function planSeededBiases(mapType) {
  const cellBiases = initBiasGrid();

  if (!ROOM_LAYOUT_TYPES.has(mapType)) {
    return cellBiases;
  }

  const settings = roomPlanningSettings();
  const roomCenters = chooseRoomLayouts(settings);
  roomCenters.forEach(room => paintRoomBiases(cellBiases, room));

  if (roomCenters.length < 2) {
    return cellBiases;
  }

  const pathDirs = initPathDirGrid();
  const doors = [];
  const connectedRooms = [roomCenters[0]];

  for (let i = 1; i < roomCenters.length; i++) {
    const room = roomCenters[i];
    let bestTarget = connectedRooms[0];
    let bestDistance = Infinity;

    connectedRooms.forEach(candidate => {
      const distance = Math.abs(candidate.x - room.x) + Math.abs(candidate.y - room.y);
      if (distance < bestDistance) {
        bestDistance = distance;
        bestTarget = candidate;
      }
    });

    const fromExit = exitForRoom(room, bestTarget);
    const toExit = exitForRoom(bestTarget, room);
    doors.push(fromExit.door, toExit.door);

    const start = inBounds(fromExit.outside.x, fromExit.outside.y) ? fromExit.outside : fromExit.door;
    const end = inBounds(toExit.outside.x, toExit.outside.y) ? toExit.outside : toExit.door;
    markPathConnection(pathDirs, start, end);
    connectedRooms.push(room);
  }

  paintDoorBiases(cellBiases, doors);
  paintCorridorBiases(cellBiases, pathDirs);
  return cellBiases;
}

// ═══════════════════════════════════════════════════════════════════════════════
// WAVE FUNCTION COLLAPSE
// ═══════════════════════════════════════════════════════════════════════════════

function wfcGenerate(mapType) {
  const typeTiles = TILES.filter(t => t.type === mapType);
  const cellBiases = planSeededBiases(mapType);

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

  function effectiveWeightFor(tileId, neighborIds, x, y) {
    const tile = TILE_BY_ID[tileId];
    if (!tile) return 0;

    const baseWeight = Number(tile.weight) || 0;
    const cellBias = (cellBiases[y] && cellBiases[y][x] && cellBiases[y][x][tileId]) || 1;
    const rules = Array.isArray(tile.neighborWeightRules) ? tile.neighborWeightRules : [];
    if (rules.length === 0 || neighborIds.length === 0) return Math.max(0, baseWeight) * cellBias;

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
    return effective * cellBias;
  }

  // Weighted random pick from an array of tile IDs with neighbor-aware overrides
  function weightedPick(ids, grid, x, y) {
    const neighborIds = collapsedNeighborIds(grid, x, y);
    const candidates = ids.map(id => ({ id, weight: effectiveWeightFor(id, neighborIds, x, y) }));

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

function pathWithEncodedSegments(path) {
  return String(path || '')
    .split('/')
    .map(segment => encodeURIComponent(segment))
    .join('/');
}

function splitTileAsset(asset) {
  const raw = String(asset || '').trim();
  const extMatch = raw.match(/\.(svg|png)$/i);
  if (!extMatch) {
    return { base: raw, ext: '' };
  }
  return {
    base: raw.slice(0, -extMatch[0].length),
    ext: extMatch[1].toLowerCase(),
  };
}

function tileUrlFor(type, assetWithExt) {
  return '/tiles/' + pathWithEncodedSegments(type) + '/' + pathWithEncodedSegments(assetWithExt);
}

function pngTileInnerMarkup(url) {
  return '<image href="' + url + '" width="' + TILE + '" height="' + TILE + '" preserveAspectRatio="none"/>';
}

async function fetchSvgInnerFromUrl(url) {
  const res = await fetch(url, { cache: 'no-store' });
  if (!res.ok) return null;
  const raw = await res.text();
  const inner = raw
    .replace(/^[\s\S]*?<svg[^>]*>/i, '')
    .replace(/<\/svg>[\s\S]*$/i, '')
    .trim();
  return inner;
}

async function getTileSVG(tile) {
  const key = tile.type + '/' + tile.asset;
  const parsedAsset = splitTileAsset(tile.asset);

  const overrideMap = (typeof window !== 'undefined' && window.ACTIVE_SVG_BY_ASSET)
    ? window.ACTIVE_SVG_BY_ASSET
    : null;
  if (overrideMap && typeof overrideMap[key] === 'string' && overrideMap[key].trim()) {
    const overrideRaw = overrideMap[key];
    // Guard against stale editor/session overrides that embed missing PNG files
    // for tiles that should use extensionless/svg-first assets.
    const overrideLooksLikePngImage = /<image[\s\S]*?href=["'][^"']+\.png["']/i.test(overrideRaw);
    if (!(overrideLooksLikePngImage && parsedAsset.ext !== 'png')) {
    const overrideInner = overrideRaw
      .replace(/^[\s\S]*?<svg[^>]*>/i, '')
      .replace(/<\/svg>[\s\S]*$/i, '')
      .trim();
    return overrideInner || overrideRaw;
    }
  }

  if (TILE_SVG_CACHE.has(key)) {
    return TILE_SVG_CACHE.get(key);
  }

  if (parsedAsset.ext === 'png') {
    const pngUrl = tileUrlFor(tile.type, tile.asset);
    const pngMarkup = pngTileInnerMarkup(pngUrl);
    TILE_SVG_CACHE.set(key, pngMarkup);
    return pngMarkup;
  }

  if (parsedAsset.ext === 'svg') {
    const svgUrl = tileUrlFor(tile.type, tile.asset);
    const inner = await fetchSvgInnerFromUrl(svgUrl);
    if (inner === null) {
      throw new Error('Failed to load tile SVG: ' + key);
    }
    TILE_SVG_CACHE.set(key, inner);
    return inner;
  }

  const svgUrl = tileUrlFor(tile.type, parsedAsset.base + '.svg');
  const svgInner = await fetchSvgInnerFromUrl(svgUrl);
  if (svgInner !== null) {
    TILE_SVG_CACHE.set(key, svgInner);
    return svgInner;
  }

  const pngUrl = tileUrlFor(tile.type, parsedAsset.base + '.png');
  const pngRes = await fetch(pngUrl, { cache: 'no-store' });
  if (!pngRes.ok) {
    throw new Error('Failed to load tile asset (svg/png): ' + key);
  }
  const pngMarkup = pngTileInnerMarkup(pngUrl);
  TILE_SVG_CACHE.set(key, pngMarkup);
  return pngMarkup;
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


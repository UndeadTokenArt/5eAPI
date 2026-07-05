// Each tile references an external asset file under /tiles/<type>/.
// `asset` may include an explicit extension (.svg or .png), or omit it for svg-first/png-fallback loading.
// Keeping metadata and assets separate lets you edit tile files without touching JS.
const TILES = [
  // ── Dungeon ──
  { id:'d-floor',    type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:10, asset:'d-floor' },
  { id:'d-wall',     type:'dungeon', edges:{N:'wall',E:'wall',S:'wall',W:'wall'},   weight:100,  asset:'d-wall' },
  
  { id:'d-corr-h',
    type:'dungeon',
    edges:{N:'wall',E:'open',S:'wall',W:'open'},
    weight:10,
    groups:['corridor','corridor'],
    neighbotWeightRules:
    [
      { tileIds:['d-corr-h'], weightOverride:0 },
      { groups:['corridor'], weightOverride:0 },
    ],
  asset:'d-corr-h' },
  
  { id:'d-corr-v',
    type:'dungeon',
    edges:{N:'open',E:'wall',S:'open',W:'wall'},
    weight:20,
    groups:['corridor','corridor'],
    neighborWeightRules:
    [
      { tileIds:['d-corr-v'], weightOverride:0 },
      { groups:['corridor'], weightOverride:0 },
    ],
  asset:'d-corr-v' },
  
  { id:'d-cross',    type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:10,  asset:'d-cross' },
  { id:'d-cor-ne',   type:'dungeon', edges:{N:'open',E:'open',S:'wall',W:'wall'},   weight:10,  asset:'d-cor-ne' },
  { id:'d-cor-se',   type:'dungeon', edges:{N:'wall',E:'open',S:'open',W:'wall'},   weight:10,  asset:'d-cor-se' },
  { id:'d-cor-sw',   type:'dungeon', edges:{N:'wall',E:'wall',S:'open',W:'open'},   weight:10,  asset:'d-cor-sw' },
  { id:'d-cor-nw',   type:'dungeon', edges:{N:'open',E:'wall',S:'wall',W:'open'},   weight:10,  asset:'d-cor-nw' },
  { id:'d-t-n',      type:'dungeon', edges:{N:'wall',E:'open',S:'open',W:'open'},   weight:10,  asset:'d-t-n' },
  { id:'d-t-e',      type:'dungeon', edges:{N:'open',E:'wall',S:'open',W:'open'},   weight:10,  asset:'d-t-e' },
  { id:'d-t-s',      type:'dungeon', edges:{N:'open',E:'open',S:'wall',W:'open'},   weight:10,  asset:'d-t-s' },
  { id:'d-t-w',      type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'wall'},   weight:10,  asset:'d-t-w' },
  { id:'d-door-h',   type:'dungeon', edges:{N:'wall',E:'open',S:'wall',W:'open'},   weight:1,  asset:'d-door-h' },
  { id:'d-door-v',   type:'dungeon', edges:{N:'open',E:'wall',S:'open',W:'wall'},   weight:1,  asset:'d-door-v' },

  // Pillars & Columns
  { id:'d-pillar',   type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:3,  asset:'d-pillar' },
  { id:'d-col-n',    type:'dungeon', edges:{N:'open',E:'wall',S:'wall',W:'wall'},   weight:1,  asset:'d-col-n' },
  { id:'d-col-e',    type:'dungeon', edges:{N:'wall',E:'open',S:'wall',W:'wall'},   weight:1,  asset:'d-col-e' },
  { id:'d-col-s',    type:'dungeon', edges:{N:'wall',E:'wall',S:'open',W:'wall'},   weight:1,  asset:'d-col-s' },
  { id:'d-col-w',    type:'dungeon', edges:{N:'wall',E:'wall',S:'wall',W:'open'},   weight:1,  asset:'d-col-w' },
  
  // Stairs
  { id:'d-stairs-u', type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-stairs-u' },
  { id:'d-stairs-d', type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-stairs-d' },
  
  // Pits & Hazards
  { id:'d-pit',      type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-pit' },
  { id:'d-trap',     type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-trap' },
  
  // Treasure & Altars
  { id:'d-chest',    type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-chest' },
  { id:'d-altar',    type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:1,  asset:'d-altar' },
  
  // Alcoves & Recesses
  { id:'d-alcove-n', type:'dungeon', edges:{N:'open',E:'wall',S:'wall',W:'wall'},   weight:3,  asset:'d-alcove-n' },
  { id:'d-alcove-e', type:'dungeon', edges:{N:'wall',E:'open',S:'wall',W:'wall'},   weight:3,  asset:'d-alcove-e' },
  { id:'d-alcove-s', type:'dungeon', edges:{N:'wall',E:'wall',S:'open',W:'wall'},   weight:3,  asset:'d-alcove-s' },
  { id:'d-alcove-w', type:'dungeon', edges:{N:'wall',E:'wall',S:'wall',W:'open'},   weight:3,  asset:'d-alcove-w' },
  
  // Water Features
  { id:'d-water',    type:'dungeon', edges:{N:'water',E:'water',S:'water',W:'water'}, weight:5,  asset:'d-water' },
  { id:'d-pool-n',   type:'dungeon', edges:{N:'water',E:'open',S:'open',W:'open'},   weight:3,  asset:'d-pool-n' },
  { id:'d-pool-e',   type:'dungeon', edges:{N:'open',E:'water',S:'open',W:'open'},   weight:3,  asset:'d-pool-e' },
  { id:'d-pool-s',   type:'dungeon', edges:{N:'open',E:'open',S:'water',W:'open'},   weight:3,  asset:'d-pool-s' },
  { id:'d-pool-w',   type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'water'},   weight:3,  asset:'d-pool-w' },
  
  // Braziers & Light Sources
  { id:'d-brazier',  type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:5,  asset:'d-brazier' },
  { id:'d-torch',    type:'dungeon', edges:{N:'open',E:'open',S:'open',W:'open'},   weight:3,  asset:'d-torch' },
  
  // Secret/Hidden
  { id:'d-secret-w', type:'dungeon', edges:{N:'wall',E:'open',S:'wall',W:'wall'},   weight:0,  asset:'d-secret-w' },
  { id:'d-secret-e', type:'dungeon', edges:{N:'wall',E:'wall',S:'wall',W:'open'},   weight:0,  asset:'d-secret-e' },

  // ── Outdoor ──
  { id:'o-grass',      type:'outdoor', edges:{N:'grass',E:'grass',S:'grass',W:'grass'},            weight:10, asset:'o-grass' },
  { id:'o-forest',     type:'outdoor', edges:{N:'forest',E:'forest',S:'forest',W:'forest'},        weight:8,  asset:'o-forest' },
  { id:'o-water',      type:'outdoor', edges:{N:'water',E:'water',S:'water',W:'water'},            weight:5,  asset:'o-water' },
  { id:'o-mountain',   type:'outdoor', edges:{N:'mountain',E:'mountain',S:'mountain',W:'mountain'}, weight:3,  asset:'o-mountain' },
  { id:'o-road-h',     type:'outdoor', edges:{N:'grass',E:'road',S:'grass',W:'road'},              weight:4,  asset:'o-road-h' },
  { id:'o-road-v',     type:'outdoor', edges:{N:'road',E:'grass',S:'road',W:'grass'},              weight:4,  asset:'o-road-v' },
  { id:'o-road-cross', type:'outdoor', edges:{N:'road',E:'road',S:'road',W:'road'},                weight:1,  asset:'o-road-cross' },
  { id:'o-road-tn',    type:'outdoor', edges:{N:'grass',E:'road',S:'road',W:'road'},               weight:2,  asset:'o-road-tn' },
  { id:'o-road-te',    type:'outdoor', edges:{N:'road',E:'grass',S:'road',W:'road'},               weight:2,  asset:'o-road-te' },
  { id:'o-road-ts',    type:'outdoor', edges:{N:'road',E:'road',S:'grass',W:'road'},               weight:2,  asset:'o-road-ts' },
  { id:'o-road-tw',    type:'outdoor', edges:{N:'road',E:'road',S:'road',W:'grass'},               weight:2,  asset:'o-road-tw' },
  { id:'o-road-c-ne',  type:'outdoor', edges:{N:'road',E:'road',S:'grass',W:'grass'},              weight:2,  asset:'o-road-c-ne' },
  { id:'o-road-c-se',  type:'outdoor', edges:{N:'grass',E:'road',S:'road',W:'grass'},              weight:2,  asset:'o-road-c-se' },
  { id:'o-road-c-sw',  type:'outdoor', edges:{N:'grass',E:'grass',S:'road',W:'road'},              weight:2,  asset:'o-road-c-sw' },
  { id:'o-road-c-nw',  type:'outdoor', edges:{N:'road',E:'grass',S:'grass',W:'road'},              weight:2,  asset:'o-road-c-nw' },
  { id:'o-shore-n',    type:'outdoor', edges:{N:'water',E:'grass',S:'grass',W:'grass'},            weight:2,  asset:'o-shore-n' },
  { id:'o-shore-e',    type:'outdoor', edges:{N:'grass',E:'water',S:'grass',W:'grass'},            weight:2,  asset:'o-shore-e' },
  { id:'o-shore-s',    type:'outdoor', edges:{N:'grass',E:'grass',S:'water',W:'grass'},            weight:2,  asset:'o-shore-s' },
  { id:'o-shore-w',    type:'outdoor', edges:{N:'grass',E:'grass',S:'grass',W:'water'},            weight:2,  asset:'o-shore-w' },
  { id:'o-shore-ne',   type:'outdoor', edges:{N:'water',E:'water',S:'grass',W:'grass'},            weight:1,  asset:'o-shore-ne' },
  { id:'o-shore-se',   type:'outdoor', edges:{N:'grass',E:'water',S:'water',W:'grass'},            weight:1,  asset:'o-shore-se' },
  { id:'o-shore-sw',   type:'outdoor', edges:{N:'grass',E:'grass',S:'water',W:'water'},            weight:1,  asset:'o-shore-sw' },
  { id:'o-shore-nw',   type:'outdoor', edges:{N:'water',E:'grass',S:'grass',W:'water'},            weight:1,  asset:'o-shore-nw' },
];

// Build lookup maps
const TILE_SET_STORAGE_KEY = 'dnd5e.activeTileSet.v1';
const TILE_BY_ID = {};
const TILE_GROUPS_BY_ID = {};
const TILE_IDS_BY_GROUP = {};
if (typeof window !== 'undefined' && !window.ACTIVE_SVG_BY_ASSET) {
  window.ACTIVE_SVG_BY_ASSET = {};
}

function normalizeTileForRuntime(tile) {
  const out = JSON.parse(JSON.stringify(tile || {}));
  out.id = String(out.id || '').trim();
  out.type = String(out.type || 'dungeon').trim();
  out.asset = String(out.asset || out.id || '').trim();

  const edges = out.edges || {};
  out.edges = {
    N: String(edges.N || 'open'),
    E: String(edges.E || 'open'),
    S: String(edges.S || 'open'),
    W: String(edges.W || 'open'),
  };

  const weight = Number(out.weight);
  out.weight = Number.isFinite(weight) && weight >= 0 ? weight : 1;
  out.groups = Array.isArray(out.groups)
    ? out.groups.map(group => String(group).trim()).filter(Boolean)
    : [];

  const rules = Array.isArray(out.neighborWeightRules)
    ? out.neighborWeightRules
    : (Array.isArray(out.neighbotWeightRules) ? out.neighbotWeightRules : []);
  out.neighborWeightRules = rules;
  delete out.neighbotWeightRules;
  return out;
}

function rebuildTileLookups() {
  Object.keys(TILE_BY_ID).forEach(key => { delete TILE_BY_ID[key]; });
  Object.keys(TILE_GROUPS_BY_ID).forEach(key => { delete TILE_GROUPS_BY_ID[key]; });
  Object.keys(TILE_IDS_BY_GROUP).forEach(key => { delete TILE_IDS_BY_GROUP[key]; });

  for (let i = 0; i < TILES.length; i += 1) {
    const tile = normalizeTileForRuntime(TILES[i]);
    TILES[i] = tile;
    TILE_BY_ID[tile.id] = tile;

    TILE_GROUPS_BY_ID[tile.id] = tile.groups;
    tile.groups.forEach(group => {
      if (!TILE_IDS_BY_GROUP[group]) TILE_IDS_BY_GROUP[group] = [];
      TILE_IDS_BY_GROUP[group].push(tile.id);
    });
  }
}

function applyActiveTileSetBundle(bundle) {
  if (!bundle || !Array.isArray(bundle.tiles)) return false;
  const normalizedTiles = bundle.tiles
    .map(normalizeTileForRuntime)
    .filter(tile => tile.id && tile.asset);
  if (normalizedTiles.length === 0) return false;

  TILES.splice(0, TILES.length, ...normalizedTiles);
  rebuildTileLookups();

  if (typeof window !== 'undefined') {
    const rawSvgMap = bundle.svgByAsset && typeof bundle.svgByAsset === 'object'
      ? bundle.svgByAsset
      : {};
    const nextSvgMap = {};
    Object.keys(rawSvgMap).forEach(key => {
      if (typeof rawSvgMap[key] === 'string') nextSvgMap[key] = rawSvgMap[key];
    });
    window.ACTIVE_SVG_BY_ASSET = nextSvgMap;
  }

  return true;
}

function saveActiveTileSetToStorage(bundle) {
  if (typeof window === 'undefined' || !window.localStorage) return false;
  try {
    window.localStorage.setItem(TILE_SET_STORAGE_KEY, JSON.stringify(bundle));
    return true;
  } catch (_) {
    return false;
  }
}

function loadActiveTileSetFromStorage() {
  if (typeof window === 'undefined' || !window.localStorage) return false;
  try {
    const raw = window.localStorage.getItem(TILE_SET_STORAGE_KEY);
    if (!raw) return false;
    const parsed = JSON.parse(raw);
    return applyActiveTileSetBundle(parsed);
  } catch (_) {
    return false;
  }
}

rebuildTileLookups();

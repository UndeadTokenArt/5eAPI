const THEMES = [
  {
    id: 'stone',
    name: 'Stone Dungeon',
    swatch: '#4a4540',
    vars: {
      '--tile-floor': '#3a3832',
      '--tile-wall': '#1a1816',
      '--tile-accent': '#5a5248',
      '--tile-void': '#0e0c0a',
      '--water-color': '#1e3a5a',
      '--grass-color': '#2a4a22',
      '--forest-color': '#1a3a18',
      '--road-color': '#5a4a32',
      '--mountain-color': '#4a4040',
      '--door-color': '#8a6d2a',
      '--grid-line': 'rgba(255,255,255,0.06)',
    }
  },
  {
    id: 'lava',
    name: 'Lava Dungeon',
    swatch: '#8b2500',
    vars: {
      '--tile-floor': '#2a1508',
      '--tile-wall': '#0e0804',
      '--tile-accent': '#c84a00',
      '--tile-void': '#050200',
      '--water-color': '#8b2500',
      '--grass-color': '#3a1a08',
      '--forest-color': '#5a2a00',
      '--road-color': '#4a2800',
      '--mountain-color': '#2a1808',
      '--door-color': '#e06000',
      '--grid-line': 'rgba(255,100,0,0.1)',
    }
  },
  {
    id: 'ice',
    name: 'Ice Dungeon',
    swatch: '#8ab8d8',
    vars: {
      '--tile-floor': '#c8dce8',
      '--tile-wall': '#788a98',
      '--tile-accent': '#a8c8e0',
      '--tile-void': '#2a3840',
      '--water-color': '#4898c8',
      '--grass-color': '#8ab8a0',
      '--forest-color': '#4a8870',
      '--road-color': '#9aacb8',
      '--mountain-color': '#6888a0',
      '--door-color': '#5ab8d8',
      '--grid-line': 'rgba(200,230,255,0.15)',
    }
  },
  {
    id: 'shadow',
    name: 'Shadow Dungeon',
    swatch: '#3a1850',
    vars: {
      '--tile-floor': '#1a0a28',
      '--tile-wall': '#080410',
      '--tile-accent': '#6828a8',
      '--tile-void': '#020008',
      '--water-color': '#180838',
      '--grass-color': '#200828',
      '--forest-color': '#180620',
      '--road-color': '#2a1040',
      '--mountain-color': '#1a0830',
      '--door-color': '#9848e0',
      '--grid-line': 'rgba(150,80,255,0.1)',
    }
  },
  {
    id: 'forest',
    name: 'Forest',
    swatch: '#2d6a1e',
    vars: {
      '--tile-floor': '#4a7838',
      '--tile-wall': '#2a3820',
      '--tile-accent': '#6aaa48',
      '--tile-void': '#1a2810',
      '--water-color': '#2858a8',
      '--grass-color': '#4a8832',
      '--forest-color': '#1e5018',
      '--road-color': '#8a6a48',
      '--mountain-color': '#6a7858',
      '--door-color': '#b88848',
      '--grid-line': 'rgba(100,180,60,0.1)',
    }
  },
  {
    id: 'desert',
    name: 'Desert',
    swatch: '#c8983a',
    vars: {
      '--tile-floor': '#c8a868',
      '--tile-wall': '#7a5830',
      '--tile-accent': '#e8c880',
      '--tile-void': '#4a3018',
      '--water-color': '#3878b8',
      '--grass-color': '#98a848',
      '--forest-color': '#5a7030',
      '--road-color': '#b8903a',
      '--mountain-color': '#987860',
      '--door-color': '#c87830',
      '--grid-line': 'rgba(200,160,80,0.15)',
    }
  },
  {
    id: 'aquatic',
    name: 'Aquatic',
    swatch: '#0868a8',
    vars: {
      '--tile-floor': '#1848a8',
      '--tile-wall': '#082848',
      '--tile-accent': '#28c8e8',
      '--tile-void': '#020c18',
      '--water-color': '#0848a0',
      '--grass-color': '#1888a8',
      '--forest-color': '#087868',
      '--road-color': '#288898',
      '--mountain-color': '#184878',
      '--door-color': '#48d8f8',
      '--grid-line': 'rgba(40,200,240,0.1)',
    }
  },
  {
    id: 'infernal',
    name: 'Infernal',
    swatch: '#9b1c1c',
    vars: {
      '--tile-floor': '#2a0808',
      '--tile-wall': '#0a0202',
      '--tile-accent': '#e84020',
      '--tile-void': '#000000',
      '--water-color': '#580818',
      '--grass-color': '#381010',
      '--forest-color': '#280808',
      '--road-color': '#481808',
      '--mountain-color': '#1a0808',
      '--door-color': '#e87020',
      '--grid-line': 'rgba(255,60,20,0.12)',
    }
  },
];

const EDGE_VALUES = ['open', 'wall', 'water', 'grass', 'road', 'mountain'];
const TILE_SIZE = 40;
const PREVIEW_ZOOM_MIN = 0.5;
const PREVIEW_ZOOM_MAX = 8;
const SHAPE_TEMPLATES = {
  rect: {
    tag: 'rect',
    attrs: { x: '4', y: '4', width: '32', height: '32', rx: '2', fill: 'var(--tile-accent)', stroke: 'none', 'stroke-width': '1', opacity: '1' }
  },
  ellipse: {
    tag: 'ellipse',
    attrs: { cx: '20', cy: '20', rx: '12', ry: '8', fill: 'var(--tile-accent)', stroke: 'var(--tile-wall)', 'stroke-width': '1', opacity: '1' }
  },
  circle: {
    tag: 'circle',
    attrs: { cx: '20', cy: '20', r: '8', fill: 'var(--tile-accent)', stroke: 'var(--tile-wall)', 'stroke-width': '1', opacity: '1' }
  },
  line: {
    tag: 'line',
    attrs: { x1: '4', y1: '4', x2: '36', y2: '36', stroke: 'var(--tile-wall)', 'stroke-width': '2', 'stroke-linecap': 'round', opacity: '1' }
  },
  polygon: {
    tag: 'polygon',
    attrs: { points: '20,4 36,36 4,36', fill: 'var(--tile-accent)', stroke: 'var(--tile-wall)', 'stroke-width': '1', opacity: '1' }
  }
};

const editorState = {
  currentId: '',
  draft: null,
  source: '',
  dirty: false,
  themeId: 'stone',
  previewZoom: 1,
  svgByAssetKey: {},
};

function cloneSvgByAssetMap(source) {
  const out = {};
  Object.keys(source || {}).forEach(key => {
    if (typeof source[key] === 'string') out[key] = source[key];
  });
  return out;
}

function syncActiveTileSetToSite(statusMessage) {
  if (typeof saveActiveTileSetToStorage !== 'function') return;
  const bundle = {
    version: 1,
    tileSize: TILE_SIZE,
    updatedAt: new Date().toISOString(),
    tiles: TILES.map(tile => ensureTileShape(tile)),
    svgByAsset: cloneSvgByAssetMap(editorState.svgByAssetKey),
  };
  saveActiveTileSetToStorage(bundle);
  if (statusMessage) {
    document.getElementById('editor-help').textContent = statusMessage;
  }
}

function makeBlankTileSvg() {
  return '<svg xmlns="http://www.w3.org/2000/svg" width="' + TILE_SIZE + '" height="' + TILE_SIZE + '" viewBox="0 0 ' + TILE_SIZE + ' ' + TILE_SIZE + '">\n' +
    '  <rect width="' + TILE_SIZE + '" height="' + TILE_SIZE + '" fill="var(--tile-floor)"/>\n' +
    '</svg>';
}

const blankSvg = makeBlankTileSvg();

function cloneTile(tile) {
  return JSON.parse(JSON.stringify(tile));
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function normalizeNameFragment(value) {
  return String(value || '')
    .toLowerCase()
    .replace(/\.svg$/i, '')
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '') || 'tile';
}

function tilePrefix(type) {
  if (!type || type.length === 0) return 't';
  return type.charAt(0).toLowerCase();
}

function nextUniqueId(type, baseName) {
  const prefix = tilePrefix(type);
  const normalizedBase = normalizeNameFragment(baseName);
  let candidate = prefix + '-' + normalizedBase;
  let i = 2;
  while (TILE_BY_ID[candidate]) {
    candidate = prefix + '-' + normalizedBase + '-' + i;
    i += 1;
  }
  return candidate;
}

function nextUniqueAsset(type, baseName) {
  const normalizedBase = normalizeNameFragment(baseName);
  let candidate = normalizedBase;
  let i = 2;
  const used = new Set(TILES.filter(t => t.type === type).map(t => t.asset));
  while (used.has(candidate)) {
    candidate = normalizedBase + '-' + i;
    i += 1;
  }
  return candidate;
}

function clearObject(obj) {
  Object.keys(obj).forEach(key => { delete obj[key]; });
}

function ensureTileShape(tile) {
  const draft = cloneTile(tile || {});
  draft.id = String(draft.id || '').trim();
  draft.type = String(draft.type || 'dungeon').trim();
  draft.asset = String(draft.asset || draft.id || 'new-tile').trim();

  const baseEdges = draft.edges || {};
  draft.edges = {
    N: String(baseEdges.N || 'open'),
    E: String(baseEdges.E || 'open'),
    S: String(baseEdges.S || 'open'),
    W: String(baseEdges.W || 'open'),
  };

  const rawWeight = Number(draft.weight);
  draft.weight = Number.isFinite(rawWeight) && rawWeight >= 0 ? rawWeight : 1;

  draft.groups = Array.isArray(draft.groups)
    ? draft.groups.map(group => String(group).trim()).filter(Boolean)
    : [];

  const rawRules = Array.isArray(draft.neighborWeightRules)
    ? draft.neighborWeightRules
    : (Array.isArray(draft.neighbotWeightRules) ? draft.neighbotWeightRules : []);

  draft.neighborWeightRules = rawRules
    .map(rule => {
      const out = {};
      if (Array.isArray(rule.tileIds)) {
        out.tileIds = rule.tileIds.map(id => String(id).trim()).filter(Boolean);
      }
      if (Array.isArray(rule.groups)) {
        out.groups = rule.groups.map(group => String(group).trim()).filter(Boolean);
      }
      const override = Number(rule.weightOverride);
      if (Number.isFinite(override)) {
        out.weightOverride = override;
      }
      return out;
    })
    .filter(rule => Number.isFinite(rule.weightOverride));

  delete draft.neighbotWeightRules;
  return draft;
}

function rebuildTileLookups() {
  clearObject(TILE_BY_ID);
  clearObject(TILE_GROUPS_BY_ID);
  clearObject(TILE_IDS_BY_GROUP);

  for (let i = 0; i < TILES.length; i += 1) {
    const normalized = ensureTileShape(TILES[i]);
    TILES[i] = normalized;
    TILE_BY_ID[normalized.id] = normalized;

    TILE_GROUPS_BY_ID[normalized.id] = normalized.groups;
    normalized.groups.forEach(group => {
      if (!TILE_IDS_BY_GROUP[group]) TILE_IDS_BY_GROUP[group] = [];
      TILE_IDS_BY_GROUP[group].push(normalized.id);
    });
  }
}

function uniqueTypes() {
  return Array.from(new Set(TILES.map(tile => tile.type))).sort();
}

function currentTheme() {
  return THEMES.find(theme => theme.id === editorState.themeId) || THEMES[0];
}

function updateStatus(message, tone) {
  const status = document.getElementById('editor-status');
  status.textContent = message;
  status.style.color = tone === 'danger' ? 'var(--danger)' : tone === 'ok' ? 'var(--ok)' : 'var(--muted)';
}

function setDirty(flag, message) {
  editorState.dirty = flag;
  updateStatus(message || (flag ? 'Unsaved changes' : 'Ready'), flag ? 'danger' : 'ok');
}

function applyEditorTheme(theme) {
  editorState.themeId = theme.id;
  document.querySelectorAll('.theme-swatch').forEach(el => {
    el.classList.toggle('active', el.dataset.themeId === theme.id);
  });
  renderPreview(editorState.source);
}

function buildEditorThemeSwatches() {
  const container = document.getElementById('editor-theme-swatches');
  container.innerHTML = THEMES.map(theme =>
    '<div class="theme-swatch' + (theme.id === editorState.themeId ? ' active' : '') + '" ' +
    'data-theme-id="' + theme.id + '" ' +
    'style="background:' + theme.swatch + ';" ' +
    'title="' + escapeHtml(theme.name) + '"></div>'
  ).join('');

  container.querySelectorAll('.theme-swatch').forEach(el => {
    el.addEventListener('click', () => {
      const theme = THEMES.find(item => item.id === el.dataset.themeId);
      if (theme) applyEditorTheme(theme);
    });
  });
}

function downloadText(filename, content, mimeType) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

function parseCsv(value) {
  return value.split(',').map(item => item.trim()).filter(Boolean);
}

function safeJsonParse(value, fallback) {
  const trimmed = String(value || '').trim();
  if (!trimmed) return fallback;
  try {
    return JSON.parse(trimmed);
  } catch (_) {
    return fallback;
  }
}

function tileKey(type, asset) {
  return String(type || '') + '/' + String(asset || '');
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

function pngPreviewSvg(url) {
  return '<svg xmlns="http://www.w3.org/2000/svg" width="' + TILE_SIZE + '" height="' + TILE_SIZE + '" viewBox="0 0 ' + TILE_SIZE + ' ' + TILE_SIZE + '">\n' +
    '  <image href="' + url + '" width="' + TILE_SIZE + '" height="' + TILE_SIZE + '" preserveAspectRatio="none"/>\n' +
    '</svg>';
}

function tileKeyFromTile(tile) {
  return tileKey(tile.type, tile.asset);
}

function defaultNewTile(type) {
  const baseType = type || 'dungeon';
  const id = nextUniqueId(baseType, 'new-tile');
  const asset = nextUniqueAsset(baseType, id);
  return {
    id: id,
    type: baseType,
    edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
    weight: 1,
    asset: asset,
    groups: [],
    neighborWeightRules: [],
  };
}

function populateTypeSelects() {
  const types = uniqueTypes();
  const typeFilter = document.getElementById('type-filter');
  const tileType = document.getElementById('tile-type');
  const oldFilter = typeFilter.value;
  const oldType = tileType.value;

  const allOptions = ['<option value="all">All types</option>']
    .concat(types.map(type => '<option value="' + escapeHtml(type) + '">' + escapeHtml(type) + '</option>'))
    .join('');

  typeFilter.innerHTML = allOptions;
  tileType.innerHTML = types.map(type => '<option value="' + escapeHtml(type) + '">' + escapeHtml(type) + '</option>').join('');

  typeFilter.value = types.includes(oldFilter) || oldFilter === 'all' ? oldFilter : 'all';
  tileType.value = types.includes(oldType) ? oldType : (types[0] || 'dungeon');
}

function populateEdgeSelects() {
  const edgeIds = ['edge-n', 'edge-e', 'edge-s', 'edge-w'];
  edgeIds.forEach(id => {
    const select = document.getElementById(id);
    select.innerHTML = EDGE_VALUES.map(value => '<option value="' + value + '">' + value + '</option>').join('');
  });
}

function renderTileList() {
  const query = document.getElementById('tile-filter').value.trim().toLowerCase();
  const typeFilter = document.getElementById('type-filter').value;
  const tiles = TILES.filter(tile => {
    const matchesType = typeFilter === 'all' || tile.type === typeFilter;
    const matchesQuery = !query || [tile.id, tile.asset, tile.type].join(' ').toLowerCase().includes(query);
    return matchesType && matchesQuery;
  });

  document.getElementById('tile-count').textContent = tiles.length + ' / ' + TILES.length;
  const list = document.getElementById('tile-list');
  list.innerHTML = tiles.map(tile =>
    '<button class="tile-item' + (tile.id === editorState.currentId ? ' active' : '') + '" data-tile-id="' + escapeHtml(tile.id) + '" type="button">' +
      '<strong>' + escapeHtml(tile.id) + '</strong>' +
      '<span>' + escapeHtml(tile.type) + ' · weight ' + escapeHtml(tile.weight) + ' · ' + escapeHtml(tile.asset) + '</span>' +
    '</button>'
  ).join('');

  list.querySelectorAll('[data-tile-id]').forEach(button => {
    button.addEventListener('click', () => loadTile(button.dataset.tileId));
  });
}

function loadTileToForm(tile) {
  document.getElementById('tile-id').value = tile.id || '';
  document.getElementById('tile-type').value = tile.type || 'dungeon';
  document.getElementById('tile-asset').value = tile.asset || tile.id || '';
  document.getElementById('tile-weight').value = tile.weight ?? 0;
  document.getElementById('edge-n').value = tile.edges.N || 'open';
  document.getElementById('edge-e').value = tile.edges.E || 'open';
  document.getElementById('edge-s').value = tile.edges.S || 'open';
  document.getElementById('edge-w').value = tile.edges.W || 'open';
  document.getElementById('tile-groups').value = tile.groups.join(', ');
  document.getElementById('tile-rules').value = JSON.stringify(tile.neighborWeightRules || [], null, 2);
}

function readFormIntoDraft() {
  if (!editorState.draft) return null;

  const parsedRules = safeJsonParse(document.getElementById('tile-rules').value, null);
  if (parsedRules === null) {
    updateStatus('Neighbor rules must be valid JSON', 'danger');
    return null;
  }

  const draft = cloneTile(editorState.draft);
  draft.id = document.getElementById('tile-id').value.trim();
  draft.type = document.getElementById('tile-type').value;
  draft.asset = document.getElementById('tile-asset').value.trim();
  draft.weight = Number(document.getElementById('tile-weight').value || 0);
  draft.edges = {
    N: document.getElementById('edge-n').value,
    E: document.getElementById('edge-e').value,
    S: document.getElementById('edge-s').value,
    W: document.getElementById('edge-w').value,
  };
  draft.groups = parseCsv(document.getElementById('tile-groups').value);
  draft.neighborWeightRules = Array.isArray(parsedRules) ? parsedRules : [];

  if (!draft.id) {
    updateStatus('Tile id is required', 'danger');
    return null;
  }
  if (!draft.asset) {
    updateStatus('Asset name is required', 'danger');
    return null;
  }

  return ensureTileShape(draft);
}

function renderPreview(svgText) {
  const frame = document.getElementById('svg-preview');
  const theme = currentTheme();
  const themeCss = Object.entries(theme.vars).map(([key, value]) => key + ':' + value + ';').join(' ');
  const body = svgText && String(svgText).trim() ? svgText : blankSvg;
  const previewZoom = Number(editorState.previewZoom) || 1;
  frame.srcdoc = '<!doctype html><html><head><meta charset="utf-8"><style>' +
    'html,body{margin:0;width:100%;height:100%;background:' + theme.vars['--tile-void'] + ';overflow:hidden;}' +
    'body{display:flex;align-items:center;justify-content:center;}' +
    '.zoom-stage{transform:scale(' + previewZoom + ');transform-origin:center center;display:inline-flex;align-items:center;justify-content:center;}' +
    'svg{max-width:100%;max-height:100%;background:' + theme.vars['--tile-void'] + ';' + themeCss + '}' +
    '</style></head><body><div class="zoom-stage">' + body + '</div></body></html>';
}

function setPreviewZoom(z) {
  const nextZoom = Math.max(PREVIEW_ZOOM_MIN, Math.min(PREVIEW_ZOOM_MAX, z));
  editorState.previewZoom = nextZoom;
  const label = document.getElementById('preview-zoom-label');
  if (label) {
    label.textContent = Math.round(nextZoom * 100) + '%';
  }
  renderPreview(editorState.source || document.getElementById('svg-source').value || blankSvg);
}

function zoomPreviewIn() {
  setPreviewZoom((Number(editorState.previewZoom) || 1) + 1);
}

function zoomPreviewOut() {
  setPreviewZoom((Number(editorState.previewZoom) || 1) - 1);
}

function zoomPreviewReset() {
  setPreviewZoom(1);
}

async function loadSvgSourceForTile(tile) {
  const key = tileKeyFromTile(tile);
  if (editorState.svgByAssetKey[key]) {
    return editorState.svgByAssetKey[key];
  }

  try {
    const parsedAsset = splitTileAsset(tile.asset);

    if (parsedAsset.ext === 'png') {
      const pngUrl = tileUrlFor(tile.type, tile.asset);
      const pngRes = await fetch(pngUrl, { cache: 'no-store' });
      if (!pngRes.ok) throw new Error('HTTP ' + pngRes.status);
      const previewSvg = pngPreviewSvg(pngUrl);
      editorState.svgByAssetKey[key] = previewSvg;
      return previewSvg;
    }

    if (parsedAsset.ext === 'svg') {
      const svgUrl = tileUrlFor(tile.type, tile.asset);
      const svgRes = await fetch(svgUrl, { cache: 'no-store' });
      if (!svgRes.ok) throw new Error('HTTP ' + svgRes.status);
      const svgText = await svgRes.text();
      editorState.svgByAssetKey[key] = svgText;
      return svgText;
    }

    const svgUrl = tileUrlFor(tile.type, parsedAsset.base + '.svg');
    const svgRes = await fetch(svgUrl, { cache: 'no-store' });
    if (svgRes.ok) {
      const svgText = await svgRes.text();
      editorState.svgByAssetKey[key] = svgText;
      return svgText;
    }

    const pngUrl = tileUrlFor(tile.type, parsedAsset.base + '.png');
    const pngRes = await fetch(pngUrl, { cache: 'no-store' });
    if (pngRes.ok) {
      const previewSvg = pngPreviewSvg(pngUrl);
      editorState.svgByAssetKey[key] = previewSvg;
      return previewSvg;
    }

    throw new Error('HTTP ' + svgRes.status + '/' + pngRes.status);
  } catch (_) {
    return blankSvg;
  }
}

async function loadTile(tileId) {
  const tile = TILE_BY_ID[tileId];
  if (!tile) return;

  editorState.currentId = tileId;
  editorState.draft = ensureTileShape(tile);
  loadTileToForm(editorState.draft);
  renderTileList();

  document.getElementById('editor-help').textContent = 'Loading SVG source...';
  const source = await loadSvgSourceForTile(editorState.draft);
  editorState.source = source;
  document.getElementById('svg-source').value = source;
  renderPreview(source);
  setDirty(false, 'Loaded ' + tile.id);
}

function persistDraftToLibrary() {
  const draft = readFormIntoDraft();
  if (!draft) return null;

  const source = document.getElementById('svg-source').value || blankSvg;
  const previousId = editorState.currentId;
  const existingIndex = TILES.findIndex(tile => tile.id === previousId);

  if (existingIndex >= 0) {
    TILES[existingIndex] = draft;
  } else {
    TILES.push(draft);
  }

  rebuildTileLookups();
  editorState.currentId = draft.id;
  editorState.draft = draft;
  editorState.source = source;
  editorState.svgByAssetKey[tileKeyFromTile(draft)] = source;
  return draft;
}

function createNewTile() {
  const selectedType = document.getElementById('tile-type').value || 'dungeon';
  editorState.currentId = 'new-tile';
  editorState.draft = ensureTileShape(defaultNewTile(selectedType));
  editorState.source = makeBlankTileSvg();
  loadTileToForm(editorState.draft);
  document.getElementById('svg-source').value = editorState.source;
  renderPreview(editorState.source);
  renderTileList();
  setDirty(true, 'New tile draft created with 40x40 boilerplate');
}

function downloadCurrentSvg() {
  const draft = readFormIntoDraft();
  if (!draft) return;
  const filename = (draft.asset || draft.id || 'tile') + '.svg';
  const source = document.getElementById('svg-source').value || blankSvg;
  downloadText(filename, source, 'image/svg+xml;charset=utf-8');
  setDirty(false, 'SVG exported as ' + filename);
}

function downloadCurrentManifest() {
  const draft = readFormIntoDraft();
  if (!draft) return;
  const source = document.getElementById('svg-source').value || blankSvg;
  const bundle = {
    version: 1,
    tile: draft,
    svg: source,
    exportedAt: new Date().toISOString(),
  };
  const filename = (draft.asset || draft.id || 'tile') + '.json';
  downloadText(filename, JSON.stringify(bundle, null, 2), 'application/json;charset=utf-8');
  setDirty(false, 'Bundle exported as ' + filename);
}

async function saveAllTilesBundle() {
  const currentDraft = persistDraftToLibrary();
  if (!currentDraft) return;

  const tilesOut = TILES.map(tile => ensureTileShape(tile));
  const svgByAsset = {};

  for (const tile of tilesOut) {
    const key = tileKeyFromTile(tile);
    let source = editorState.svgByAssetKey[key];
    if (!source) {
      source = await loadSvgSourceForTile(tile);
      editorState.svgByAssetKey[key] = source;
    }
    svgByAsset[key] = source;
  }

  const bundle = {
    version: 1,
    tileSize: TILE_SIZE,
    exportedAt: new Date().toISOString(),
    tiles: tilesOut,
    svgByAsset: svgByAsset,
  };

  downloadText('tile-editor-all.json', JSON.stringify(bundle, null, 2), 'application/json;charset=utf-8');
  syncActiveTileSetToSite('Active tile set synced for current site session');
  setDirty(false, 'Saved all tiles and metadata to tile-editor-all.json');
}

async function importSvgFiles(files) {
  if (!files || files.length === 0) return;

  const selectedType = document.getElementById('tile-type').value || 'dungeon';
  const imported = [];

  for (const file of files) {
    const fileName = file.name || 'tile.svg';
    const baseName = normalizeNameFragment(fileName);
    const asset = nextUniqueAsset(selectedType, baseName);
    const id = nextUniqueId(selectedType, asset);
    const source = await file.text();

    const tile = ensureTileShape({
      id: id,
      type: selectedType,
      edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
      weight: 1,
      asset: asset,
      groups: [],
      neighborWeightRules: [],
    });

    TILES.push(tile);
    editorState.svgByAssetKey[tileKeyFromTile(tile)] = source || blankSvg;
    imported.push(tile.id);
  }

  rebuildTileLookups();
  populateTypeSelects();
  renderTileList();

  if (imported.length > 0) {
    await loadTile(imported[imported.length - 1]);
  }
  syncActiveTileSetToSite('Active tile set updated from imported SVG files');
  setDirty(true, 'Imported ' + imported.length + ' SVG file(s)');
}

async function handleSvgImportInput(event) {
  const input = event.target;
  const files = Array.from(input.files || []);
  await importSvgFiles(files);
  input.value = '';
}

async function loadTilesBundle(file) {
  const text = await file.text();
  const payload = safeJsonParse(text, null);
  if (!payload || !Array.isArray(payload.tiles)) {
    updateStatus('Selected JSON does not contain a valid tiles array', 'danger');
    return;
  }

  const normalizedTiles = payload.tiles.map(tile => ensureTileShape(tile)).filter(tile => tile.id && tile.asset);
  if (normalizedTiles.length === 0) {
    updateStatus('Loaded JSON had zero valid tiles', 'danger');
    return;
  }

  TILES.splice(0, TILES.length, ...normalizedTiles);
  rebuildTileLookups();

  editorState.svgByAssetKey = {};
  if (payload.svgByAsset && typeof payload.svgByAsset === 'object') {
    Object.keys(payload.svgByAsset).forEach(key => {
      if (typeof payload.svgByAsset[key] === 'string') {
        editorState.svgByAssetKey[key] = payload.svgByAsset[key];
      }
    });
  }

  populateTypeSelects();
  populateEdgeSelects();
  renderTileList();
  await loadTile(normalizedTiles[0].id);
  syncActiveTileSetToSite('Active tile set updated from loaded JSON bundle');
  setDirty(false, 'Loaded ' + normalizedTiles.length + ' tiles from JSON bundle');
}

async function handleJsonLoadInput(event) {
  const input = event.target;
  const file = (input.files || [])[0];
  if (!file) return;
  await loadTilesBundle(file);
  input.value = '';
}

function buildShapeTemplateControls() {
  const typeSelect = document.getElementById('shape-template-type');
  typeSelect.innerHTML = Object.keys(SHAPE_TEMPLATES)
    .map(name => '<option value="' + name + '">' + name + '</option>')
    .join('');
  renderShapeTemplateFields(typeSelect.value || 'rect');
}

function renderShapeTemplateFields(shapeType) {
  const template = SHAPE_TEMPLATES[shapeType] || SHAPE_TEMPLATES.rect;
  const container = document.getElementById('shape-template-fields');
  container.innerHTML = Object.entries(template.attrs).map(([attr, value]) =>
    '<label class="shape-template-field">' +
      '<span>' + escapeHtml(attr) + '</span>' +
      '<input type="text" data-shape-attr="' + escapeHtml(attr) + '" value="' + escapeHtml(value) + '">' +
    '</label>'
  ).join('');
}

function insertShapeTemplateTag() {
  const shapeType = document.getElementById('shape-template-type').value;
  const template = SHAPE_TEMPLATES[shapeType] || SHAPE_TEMPLATES.rect;
  const attrs = [];
  document.querySelectorAll('#shape-template-fields [data-shape-attr]').forEach(input => {
    const name = input.getAttribute('data-shape-attr');
    const value = input.value;
    if (name && String(value).trim() !== '') {
      attrs.push(name + '="' + String(value).replace(/"/g, '&quot;') + '"');
    }
  });

  const tag = '  <' + template.tag + (attrs.length ? ' ' + attrs.join(' ') : '') + ' />';
  const textarea = document.getElementById('svg-source');
  let source = textarea.value || makeBlankTileSvg();

  if (source.includes('</svg>')) {
    source = source.replace('</svg>', tag + '\n</svg>');
  } else {
    source += '\n' + tag;
  }

  textarea.value = source;
  editorState.source = source;
  renderPreview(source);
  setDirty(true, template.tag + ' template inserted');
}

function applyChanges() {
  const draft = persistDraftToLibrary();
  if (!draft) return;
  syncActiveTileSetToSite('Active tile set synced from editor changes');
  renderTileList();
  setDirty(true, 'Draft updated');
}

function resetEditor() {
  if (!editorState.currentId) return;
  loadTile(editorState.currentId);
}

function reloadSource() {
  if (!editorState.currentId) return;
  loadTile(editorState.currentId);
}

function bindEditorEvents() {
  document.getElementById('tile-filter').addEventListener('input', renderTileList);
  document.getElementById('type-filter').addEventListener('change', renderTileList);
  document.getElementById('shape-template-type').addEventListener('change', event => {
    renderShapeTemplateFields(event.target.value);
  });
  document.getElementById('insert-shape-btn').addEventListener('click', insertShapeTemplateTag);
  document.getElementById('preview-zoom-in').addEventListener('click', zoomPreviewIn);
  document.getElementById('preview-zoom-out').addEventListener('click', zoomPreviewOut);
  document.getElementById('preview-zoom-reset').addEventListener('click', zoomPreviewReset);

  const previewFrame = document.getElementById('preview-frame');
  if (previewFrame) {
    previewFrame.addEventListener('wheel', event => {
      event.preventDefault();
      setPreviewZoom((Number(editorState.previewZoom) || 1) + (event.deltaY < 0 ? 0.15 : -0.15));
    }, { passive: false });
  }

  document.getElementById('svg-source').addEventListener('input', () => {
    editorState.source = document.getElementById('svg-source').value;
    renderPreview(editorState.source);
    setDirty(true, 'SVG source edited');
  });

  document.getElementById('apply-btn').addEventListener('click', applyChanges);
  document.getElementById('reset-btn').addEventListener('click', resetEditor);
  document.getElementById('reload-btn').addEventListener('click', reloadSource);
  document.getElementById('new-btn').addEventListener('click', createNewTile);
  document.getElementById('svg-export-btn').addEventListener('click', downloadCurrentSvg);
  document.getElementById('json-export-btn').addEventListener('click', downloadCurrentManifest);
  document.getElementById('save-all-btn').addEventListener('click', saveAllTilesBundle);
  document.getElementById('download-manifest-btn').addEventListener('click', saveAllTilesBundle);

  document.getElementById('import-svg-btn').addEventListener('click', () => {
    document.getElementById('import-svg-input').click();
  });
  document.getElementById('load-json-btn').addEventListener('click', () => {
    document.getElementById('load-json-input').click();
  });
  document.getElementById('import-svg-input').addEventListener('change', handleSvgImportInput);
  document.getElementById('load-json-input').addEventListener('change', handleJsonLoadInput);
}

(function initEditor() {
  if (typeof loadActiveTileSetFromStorage === 'function') {
    loadActiveTileSetFromStorage();
  }
  rebuildTileLookups();
  buildEditorThemeSwatches();
  buildShapeTemplateControls();
  populateTypeSelects();
  populateEdgeSelects();
  bindEditorEvents();
  setPreviewZoom(1);
  renderTileList();

  if (TILES.length > 0) {
    loadTile(TILES[0].id);
  } else {
    editorState.draft = ensureTileShape(defaultNewTile('dungeon'));
    loadTileToForm(editorState.draft);
    editorState.source = makeBlankTileSvg();
    document.getElementById('svg-source').value = editorState.source;
    renderPreview(editorState.source);
    setDirty(false, 'No tiles found');
  }

  applyEditorTheme(THEMES[0]);
})();

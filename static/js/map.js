// ═══════════════════════════════════════════════════════════════════════════════
// CONSTANTS
// ═══════════════════════════════════════════════════════════════════════════════
const GRID = 20;
const TILE = 40;
let currentMapType = 'dungeon';
let currentTheme = null;
let currentZoom = 1.0;
let currentPropType = 'none';
let currentRoomCount = 5;
let currentRoomSizePreset = 'varied';

if (typeof window !== 'undefined') {
  window.MAP_ROOM_SETTINGS = window.MAP_ROOM_SETTINGS || {
    count: currentRoomCount,
    sizePreset: currentRoomSizePreset,
  };
}

function syncRoomSettings() {
  if (typeof window === 'undefined') return;
  window.MAP_ROOM_SETTINGS = {
    count: currentRoomCount,
    sizePreset: currentRoomSizePreset,
  };
}

function setRoomCount(value) {
  const parsed = Number(value);
  currentRoomCount = Number.isFinite(parsed) ? Math.max(1, Math.min(8, Math.round(parsed))) : 5;
  const select = document.getElementById('room-count-select');
  if (select) select.value = String(currentRoomCount);
  syncRoomSettings();
}

function setRoomSizePreset(value) {
  currentRoomSizePreset = ['compact', 'varied', 'broad'].includes(value) ? value : 'varied';
  const select = document.getElementById('room-size-select');
  if (select) select.value = currentRoomSizePreset;
  syncRoomSettings();
}

// ═══════════════════════════════════════════════════════════════════════════════
// COLOR THEMES
// ═══════════════════════════════════════════════════════════════════════════════
const THEMES = [
  {
    id: 'stone',
    name: 'Stone Dungeon',
    swatch: '#4a4540',
    vars: {
      '--tile-floor':    '#3a3832',
      '--tile-wall':     '#1a1816',
      '--tile-accent':   '#5a5248',
      '--tile-void':     '#0e0c0a',
      '--water-color':   '#1e3a5a',
      '--grass-color':   '#2a4a22',
      '--forest-color':  '#1a3a18',
      '--road-color':    '#5a4a32',
      '--mountain-color':'#4a4040',
      '--door-color':    '#8a6d2a',
      '--grid-line':     'rgba(255,255,255,0.06)',
    }
  },
  {
    id: 'lava',
    name: 'Lava Dungeon',
    swatch: '#8b2500',
    vars: {
      '--tile-floor':    '#2a1508',
      '--tile-wall':     '#0e0804',
      '--tile-accent':   '#c84a00',
      '--tile-void':     '#050200',
      '--water-color':   '#8b2500',
      '--grass-color':   '#3a1a08',
      '--forest-color':  '#5a2a00',
      '--road-color':    '#4a2800',
      '--mountain-color':'#2a1808',
      '--door-color':    '#e06000',
      '--grid-line':     'rgba(255,100,0,0.1)',
    }
  },
  {
    id: 'ice',
    name: 'Ice Dungeon',
    swatch: '#8ab8d8',
    vars: {
      '--tile-floor':    '#c8dce8',
      '--tile-wall':     '#788a98',
      '--tile-accent':   '#a8c8e0',
      '--tile-void':     '#2a3840',
      '--water-color':   '#4898c8',
      '--grass-color':   '#8ab8a0',
      '--forest-color':  '#4a8870',
      '--road-color':    '#9aacb8',
      '--mountain-color':'#6888a0',
      '--door-color':    '#5ab8d8',
      '--grid-line':     'rgba(200,230,255,0.15)',
    }
  },
  {
    id: 'shadow',
    name: 'Shadow Dungeon',
    swatch: '#3a1850',
    vars: {
      '--tile-floor':    '#1a0a28',
      '--tile-wall':     '#080410',
      '--tile-accent':   '#6828a8',
      '--tile-void':     '#020008',
      '--water-color':   '#180838',
      '--grass-color':   '#200828',
      '--forest-color':  '#180620',
      '--road-color':    '#2a1040',
      '--mountain-color':'#1a0830',
      '--door-color':    '#9848e0',
      '--grid-line':     'rgba(150,80,255,0.1)',
    }
  },
  {
    id: 'forest',
    name: 'Forest',
    swatch: '#2d6a1e',
    vars: {
      '--tile-floor':    '#4a7838',
      '--tile-wall':     '#2a3820',
      '--tile-accent':   '#6aaa48',
      '--tile-void':     '#1a2810',
      '--water-color':   '#2858a8',
      '--grass-color':   '#4a8832',
      '--forest-color':  '#1e5018',
      '--road-color':    '#8a6a48',
      '--mountain-color':'#6a7858',
      '--door-color':    '#b88848',
      '--grid-line':     'rgba(100,180,60,0.1)',
    }
  },
  {
    id: 'desert',
    name: 'Desert',
    swatch: '#c8983a',
    vars: {
      '--tile-floor':    '#c8a868',
      '--tile-wall':     '#7a5830',
      '--tile-accent':   '#e8c880',
      '--tile-void':     '#4a3018',
      '--water-color':   '#3878b8',
      '--grass-color':   '#98a848',
      '--forest-color':  '#5a7030',
      '--road-color':    '#b8903a',
      '--mountain-color':'#987860',
      '--door-color':    '#c87830',
      '--grid-line':     'rgba(200,160,80,0.15)',
    }
  },
  {
    id: 'aquatic',
    name: 'Aquatic',
    swatch: '#0868a8',
    vars: {
      '--tile-floor':    '#1848a8',
      '--tile-wall':     '#082848',
      '--tile-accent':   '#28c8e8',
      '--tile-void':     '#020c18',
      '--water-color':   '#0848a0',
      '--grass-color':   '#1888a8',
      '--forest-color':  '#087868',
      '--road-color':    '#288898',
      '--mountain-color':'#184878',
      '--door-color':    '#48d8f8',
      '--grid-line':     'rgba(40,200,240,0.1)',
    }
  },
  {
    id: 'infernal',
    name: 'Infernal',
    swatch: '#9b1c1c',
    vars: {
      '--tile-floor':    '#2a0808',
      '--tile-wall':     '#0a0202',
      '--tile-accent':   '#e84020',
      '--tile-void':     '#000000',
      '--water-color':   '#580818',
      '--grass-color':   '#381010',
      '--forest-color':  '#280808',
      '--road-color':    '#481808',
      '--mountain-color':'#1a0808',
      '--door-color':    '#e87020',
      '--grid-line':     'rgba(255,60,20,0.12)',
    }
  },
];

currentTheme = THEMES[0];


// ═══════════════════════════════════════════════════════════════════════════════
// MAP TYPE
// ═══════════════════════════════════════════════════════════════════════════════
function setMapType(type) {
  currentMapType = type;
  document.querySelectorAll('#map-type-buttons .type-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.mapType === type);
  });
  if (typeof buildPropTypeOptions === 'function') {
    buildPropTypeOptions(currentMapType, currentPropType);
    const select = document.getElementById('prop-type-select');
    if (select) currentPropType = select.value || 'all';
  }
}

function setPropType(type) {
  currentPropType = type || 'none';
}

async function fetchAvailableBiomes() {
  try {
    const res = await fetch('/api/map/biomes', { cache: 'no-store' });
    if (!res.ok) return [];
    const biomes = await res.json();
    return Array.isArray(biomes) ? biomes : [];
  } catch (_) {
    return [];
  }
}

function prettyBiomeName(name) {
  if (!name) return '';
  return name.charAt(0).toUpperCase() + name.slice(1).replace(/-/g, ' ');
}

function biomeIcon(name) {
  const icons = {
    dungeon: '⛏',
    outdoor: '🌿',
    forest: '🌲',
    desert: '🏜',
    aquatic: '🌊',
    infernal: '🔥',
  };
  return icons[name] || '🧭';
}

function buildMapTypeButtons(biomes) {
  const container = document.getElementById('map-type-buttons');
  const available = biomes.filter(b => TILES.some(t => t.type === b));
  const types = available.length > 0
    ? available
    : Array.from(new Set(TILES.map(t => t.type))).sort();

  if (!types.includes(currentMapType)) {
    currentMapType = types[0] || 'dungeon';
  }

  container.innerHTML = types.map(type =>
    '<button class="type-btn' + (type === currentMapType ? ' active' : '') + '" ' +
    'data-map-type="' + type + '" onclick="setMapType(\'' + type + '\')">' +
    biomeIcon(type) + ' ' + prettyBiomeName(type) +
    '</button>'
  ).join('');
}

// ═══════════════════════════════════════════════════════════════════════════════
// GENERATE
// ═══════════════════════════════════════════════════════════════════════════════
async function generate() {
  const status = document.getElementById('map-status');
  status.textContent = 'Generating…';

  // Hide placeholder
  const ph = document.getElementById('map-placeholder-fo');
  if (ph) ph.style.display = 'none';

  // Small async yield so status renders before heavy computation
  setTimeout(async () => {
    const t0 = performance.now();
    try {
      await buildTileDefs();
    } catch (err) {
      status.textContent = '⚠ Failed to load tile files: ' + err.message;
      return;
    }

    const grid = wfcGenerate(currentMapType);
    const ms = Math.round(performance.now() - t0);

    if (!grid) {
      status.textContent = '⚠ Generation failed after 8 attempts. Try again.';
      return;
    }

    renderGrid(grid);
    if (typeof buildPropDefs === 'function') {
      buildPropDefs();
    }
    if (typeof generatePropPlacements === 'function' && typeof renderPropLayer === 'function') {
      const placements = currentPropType === 'none'
        ? []
        : generatePropPlacements(grid, currentMapType, currentPropType);
      renderPropLayer(placements);
    }
    status.textContent = currentMapType.charAt(0).toUpperCase() + currentMapType.slice(1) +
      ' map generated in ' + ms + 'ms · ' + GRID + '×' + GRID + ' tiles';
  }, 10);
}

// ═══════════════════════════════════════════════════════════════════════════════
// THEMES
// ═══════════════════════════════════════════════════════════════════════════════
function applyTheme(theme) {
  currentTheme = theme;
  const svg = document.getElementById('map-svg');
  for (const [k, v] of Object.entries(theme.vars)) {
    svg.style.setProperty(k, v);
  }
  // Update swatch active state
  document.querySelectorAll('.theme-swatch').forEach(el => {
    el.classList.toggle('active', el.dataset.themeId === theme.id);
  });
  // Rebuild tile defs so SVG fill inherits new vars (inline SVG repaints)
  buildTileDefs().catch(() => {});
}

function buildThemeSwatches() {
  const container = document.getElementById('theme-swatches');
  container.innerHTML = THEMES.map(theme =>
    '<div class="theme-swatch' + (theme.id === currentTheme.id ? ' active' : '') + '" ' +
    'data-theme-id="' + theme.id + '" ' +
    'style="background:' + theme.swatch + ';" ' +
    'title="' + theme.name + '" ' +
    'onclick="applyTheme(THEMES.find(t=>t.id===\'' + theme.id + '\'))"></div>'
  ).join('');
}

// ═══════════════════════════════════════════════════════════════════════════════
// ZOOM
// ═══════════════════════════════════════════════════════════════════════════════
function setZoom(z) {
  currentZoom = Math.max(0.5, Math.min(3.0, z));
  const label = document.getElementById('zoom-label');
  const wrap = document.getElementById('zoom-wrap');
  const svg = document.getElementById('map-svg');
  if (!label || !wrap || !svg) return;

  const baseSize = GRID * TILE;
  const size = baseSize * currentZoom;
  label.textContent = Math.round(currentZoom * 100) + '%';
  wrap.style.transform = 'scale(' + currentZoom + ')';
  wrap.style.width = size + 'px';
  wrap.style.height = size + 'px';
  svg.style.width = baseSize + 'px';
  svg.style.height = baseSize + 'px';
}
function zoomIn()    { setZoom(currentZoom + 0.25); }
function zoomOut()   { setZoom(currentZoom - 0.25); }
function zoomReset() { setZoom(1.0); }

// ═══════════════════════════════════════════════════════════════════════════════
// INIT
// ═══════════════════════════════════════════════════════════════════════════════
(async function init() {
  const viewport = document.getElementById('map-viewport');
  if (viewport) {
    viewport.addEventListener('wheel', function(e) {
      e.preventDefault();
      setZoom(currentZoom + (e.deltaY < 0 ? 0.15 : -0.15));
    }, { passive: false });
  }

  if (typeof loadActiveTileSetFromStorage === 'function') {
    loadActiveTileSetFromStorage();
  }

  setRoomCount(document.getElementById('room-count-select').value);
  setRoomSizePreset(document.getElementById('room-size-select').value);

  const biomes = await fetchAvailableBiomes();
  buildMapTypeButtons(biomes);
  if (typeof buildPropTypeOptions === 'function') {
    buildPropTypeOptions(currentMapType, currentPropType);
    const select = document.getElementById('prop-type-select');
    if (select) currentPropType = select.value || 'none';
  }
  try {
    await buildTileDefs();
  } catch (err) {
    document.getElementById('map-status').textContent = '⚠ Failed to load tile files: ' + err.message;
  }
  if (typeof buildPropDefs === 'function') {
    buildPropDefs();
  }
  applyTheme(THEMES[0]);
  buildThemeSwatches();
  setZoom(1.0);
})();

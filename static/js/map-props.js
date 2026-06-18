const PROP_TEMPLATES = [
  {
    id: 'prop-vine-clump',
    propType: 'vegetation',
    allowedMapTypes: ['outdoor', 'indoor', 'castle'],
    weight: 5,
    spawnChance: 0.14,
    svg: '<path d="M6 30c4-6 9-8 15-8 5 0 9 2 13 6" fill="none" stroke="var(--forest-color)" stroke-width="2" stroke-linecap="round"/><circle cx="11" cy="24" r="2" fill="var(--grass-color)"/><circle cx="18" cy="21" r="2.2" fill="var(--grass-color)"/><circle cx="25" cy="23" r="2" fill="var(--grass-color)"/>'
  },
  {
    id: 'prop-spike-trap',
    propType: 'traps',
    allowedMapTypes: ['dungeon', 'indoor', 'castle'],
    weight: 4,
    spawnChance: 0.08,
    svg: '<polygon points="10,30 14,20 18,30" fill="var(--tile-wall)"/><polygon points="18,30 22,18 26,30" fill="var(--tile-wall)"/><polygon points="26,30 30,21 34,30" fill="var(--tile-wall)"/><rect x="9" y="30" width="26" height="2" fill="var(--tile-accent)" opacity="0.8"/>'
  },
  {
    id: 'prop-gold-cache',
    propType: 'treasure',
    allowedMapTypes: ['dungeon', 'indoor', 'castle'],
    weight: 3,
    spawnChance: 0.06,
    svg: '<rect x="10" y="20" width="20" height="10" rx="2" fill="var(--door-color)"/><rect x="10" y="24" width="20" height="2" fill="var(--tile-wall)" opacity="0.5"/><circle cx="20" cy="25" r="1.4" fill="var(--tile-void)"/><circle cx="28" cy="18" r="1.5" fill="var(--door-color)"/><circle cx="30" cy="16" r="1.2" fill="var(--door-color)"/>'
  },
  {
    id: 'prop-ruin-column',
    propType: 'ruins',
    allowedMapTypes: ['outdoor', 'castle', 'indoor'],
    weight: 4,
    spawnChance: 0.09,
    svg: '<rect x="15" y="10" width="10" height="20" fill="var(--tile-wall)" opacity="0.85"/><rect x="13" y="8" width="14" height="3" fill="var(--tile-accent)" opacity="0.8"/><rect x="13" y="30" width="14" height="2" fill="var(--tile-accent)" opacity="0.7"/>'
  },
  {
    id: 'prop-floating-wisp',
    propType: 'floating',
    allowedMapTypes: ['dungeon', 'outdoor', 'indoor', 'castle'],
    weight: 2,
    spawnChance: 0.05,
    svg: '<circle cx="20" cy="16" r="4" fill="var(--tile-accent)" opacity="0.75"/><circle cx="20" cy="16" r="2" fill="var(--tile-floor)" opacity="0.8"/><path d="M18 21c2 1 3 3 2 7" fill="none" stroke="var(--tile-accent)" stroke-width="1.2" opacity="0.8"/>'
  },
  {
    id: 'prop-flower-patch',
    propType: 'flower',
    allowedMapTypes: ['outdoor', 'indoor'],
    weight: 4,
    spawnChance: 0.12,
    svg: '<circle cx="14" cy="26" r="2" fill="var(--tile-accent)"/><circle cx="18" cy="24" r="1.8" fill="var(--tile-accent)"/><circle cx="24" cy="25" r="2" fill="var(--tile-accent)"/><path d="M14 26v5M18 24v7M24 25v6" stroke="var(--grass-color)" stroke-width="1" stroke-linecap="round"/>'
  },
];

function propTypesForMapType(mapType) {
  const set = new Set();
  PROP_TEMPLATES.forEach(template => {
    if (Array.isArray(template.allowedMapTypes) && template.allowedMapTypes.includes(mapType)) {
      set.add(template.propType);
    }
  });
  return Array.from(set).sort();
}

function prettyPropType(value) {
  if (!value || value === 'all') return 'All';
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function buildPropTypeOptions(mapType, selected) {
  const select = document.getElementById('prop-type-select');
  if (!select) return;

  const options = ['all'].concat(propTypesForMapType(mapType));
  select.innerHTML = options.map(type =>
    '<option value="' + type + '">' + prettyPropType(type) + '</option>'
  ).join('');

  const nextSelected = options.includes(selected) ? selected : 'all';
  select.value = nextSelected;
}

function buildPropDefs() {
  const defs = document.getElementById('prop-defs');
  if (!defs) return;
  const groups = PROP_TEMPLATES.map(template => '<g id="' + template.id + '">' + template.svg + '</g>');
  defs.innerHTML = groups.join('\n');
}

function tileCanHostProp(tile) {
  if (!tile) return false;
  const e = tile.edges || {};
  const walls = [e.N, e.E, e.S, e.W].filter(v => v === 'wall').length;
  return walls < 4;
}

function weightedPropPick(candidates) {
  let total = 0;
  candidates.forEach(item => { total += item.weight || 0; });
  if (total <= 0) return null;

  let r = Math.random() * total;
  for (const item of candidates) {
    r -= item.weight || 0;
    if (r <= 0) return item;
  }
  return candidates[candidates.length - 1] || null;
}

function generatePropPlacements(grid, mapType, selectedPropType) {
  const placements = [];

  const templates = PROP_TEMPLATES.filter(template => {
    const mapOk = Array.isArray(template.allowedMapTypes) && template.allowedMapTypes.includes(mapType);
    const typeOk = selectedPropType === 'all' || template.propType === selectedPropType;
    return mapOk && typeOk;
  });

  if (templates.length === 0) return placements;

  for (let y = 0; y < GRID; y++) {
    for (let x = 0; x < GRID; x++) {
      const ids = Array.from(grid[y][x]);
      if (ids.length !== 1) continue;

      const tile = TILE_BY_ID[ids[0]];
      if (!tileCanHostProp(tile)) continue;

      const rollCandidates = templates.filter(template => Math.random() < (template.spawnChance || 0));
      if (rollCandidates.length === 0) continue;

      const chosen = weightedPropPick(rollCandidates);
      if (!chosen) continue;

      const jitterX = (Math.random() * 8) - 4;
      const jitterY = (Math.random() * 8) - 4;
      const scale = 0.75 + (Math.random() * 0.45);
      const rotation = Math.round((Math.random() * 50) - 25);
      const opacity = 0.72 + (Math.random() * 0.28);

      placements.push({
        id: chosen.id,
        x,
        y,
        jitterX,
        jitterY,
        scale,
        rotation,
        opacity,
      });
    }
  }

  return placements;
}

function renderPropLayer(placements) {
  const layer = document.getElementById('prop-layer');
  if (!layer) return;

  const fragments = placements.map(prop => {
    const cx = prop.x * TILE + (TILE / 2) + prop.jitterX;
    const cy = prop.y * TILE + (TILE / 2) + prop.jitterY;
    return '<use href="#' + prop.id + '" transform="translate(' + cx + ',' + cy + ') rotate(' + prop.rotation + ') scale(' + prop.scale + ') translate(-20,-20)" opacity="' + prop.opacity.toFixed(2) + '"/>';
  });

  layer.innerHTML = fragments.join('');
}

import test from 'node:test';
import assert from 'node:assert/strict';
import { exportSymbols, loadBrowserScript } from './helpers/load-script.mjs';

test('prettyPropType handles none/all/custom types', async () => {
  const ctx = await loadBrowserScript('static/js/map-props.js');
  const { prettyPropType } = exportSymbols(ctx, ['prettyPropType']);

  assert.equal(prettyPropType('none'), 'None');
  assert.equal(prettyPropType('all'), 'All');
  assert.equal(prettyPropType('treasure'), 'Treasure');
});

test('propTypesForMapType returns sorted unique types', async () => {
  const ctx = await loadBrowserScript('static/js/map-props.js');
  const { propTypesForMapType } = exportSymbols(ctx, ['propTypesForMapType']);

  const types = propTypesForMapType('dungeon');
  assert.ok(types.includes('floating'));
  assert.ok(types.includes('traps'));
  assert.ok(types.includes('treasure'));

  const sorted = [...types].sort();
  assert.deepEqual(types, sorted);
});

test('generatePropPlacements returns no props when selected type is none', async () => {
  const ctx = await loadBrowserScript('static/js/map-props.js', {
    GRID: 2,
    TILE: 40,
    randomSequence: [0.1, 0.1, 0.1],
  });

  const grid = [
    [new Set(['d-floor']), new Set(['d-floor'])],
    [new Set(['d-floor']), new Set(['d-floor'])],
  ];
  ctx.TILE_BY_ID = {
    'd-floor': { edges: { N: 'open', E: 'open', S: 'open', W: 'open' } },
  };

  const { generatePropPlacements } = exportSymbols(ctx, ['generatePropPlacements']);
  const placements = generatePropPlacements(grid, 'dungeon', 'none');

  assert.equal(Array.isArray(placements), true);
  assert.equal(placements.length, 0);
});

test('generatePropPlacements creates deterministic placement shape', async () => {
  const ctx = await loadBrowserScript('static/js/map-props.js', {
    GRID: 1,
    TILE: 40,
    randomSequence: [
      0.01, // spawn roll (pass)
      0.0,  // weighted pick
      0.5,  // jitterX
      0.5,  // jitterY
      0.5,  // scale
      0.5,  // rotation
      0.5,  // opacity
    ],
  });

  const grid = [[new Set(['d-floor'])]];
  ctx.TILE_BY_ID = {
    'd-floor': { edges: { N: 'open', E: 'open', S: 'open', W: 'open' } },
  };

  const { generatePropPlacements } = exportSymbols(ctx, ['generatePropPlacements']);
  const placements = generatePropPlacements(grid, 'dungeon', 'all');

  assert.equal(placements.length, 1);
  const p = placements[0];
  assert.equal(typeof p.id, 'string');
  assert.equal(p.x, 0);
  assert.equal(p.y, 0);
  assert.equal(typeof p.scale, 'number');
  assert.equal(typeof p.rotation, 'number');
  assert.equal(typeof p.opacity, 'number');
});

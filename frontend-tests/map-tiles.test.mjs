import test from 'node:test';
import assert from 'node:assert/strict';
import { exportSymbols, loadBrowserScript } from './helpers/load-script.mjs';

test('normalizeTileForRuntime normalizes typo key and defaults', async () => {
  const ctx = await loadBrowserScript('static/js/map-tiles.js', { window: {} });
  const { normalizeTileForRuntime } = exportSymbols(ctx, ['normalizeTileForRuntime']);

  const out = normalizeTileForRuntime({
    id: ' x-id ',
    type: ' dungeon ',
    weight: -2,
    neighbotWeightRules: [{ groups: ['corridor'], weightOverride: 0 }],
  });

  assert.equal(out.id, 'x-id');
  assert.equal(out.type, 'dungeon');
  assert.equal(out.asset, 'x-id');
  assert.equal(out.weight, 1);
  assert.equal(out.edges.N, 'open');
  assert.equal(out.edges.E, 'open');
  assert.equal(out.edges.S, 'open');
  assert.equal(out.edges.W, 'open');
  assert.ok(Array.isArray(out.neighborWeightRules));
  assert.equal(out.neighborWeightRules.length, 1);
  assert.equal(Object.prototype.hasOwnProperty.call(out, 'neighbotWeightRules'), false);
});

test('normalizeTileForRuntime strips trailing asset extension', async () => {
  const ctx = await loadBrowserScript('static/js/map-tiles.js', { window: {} });
  const { normalizeTileForRuntime } = exportSymbols(ctx, ['normalizeTileForRuntime']);

  const out = normalizeTileForRuntime({
    id: 'd-floor',
    type: 'dungeon',
    asset: 'd-floor.png',
    edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
  });

  assert.equal(out.asset, 'd-floor');
});

test('applyActiveTileSetBundle updates tiles and ACTIVE_SVG_BY_ASSET map', async () => {
  const ctx = await loadBrowserScript('static/js/map-tiles.js', { window: {} });
  const { applyActiveTileSetBundle, TILE_BY_ID } = exportSymbols(ctx, ['applyActiveTileSetBundle', 'TILE_BY_ID']);

  const ok = applyActiveTileSetBundle({
    tiles: [
      {
        id: 't-floor',
        type: 'dungeon',
        edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
        weight: 2,
        asset: 't-floor',
      },
    ],
    svgByAsset: {
      'dungeon/t-floor': '<svg><rect/></svg>',
      'dungeon/not-string': 42,
    },
  });

  assert.equal(ok, true);
  assert.ok(TILE_BY_ID['t-floor']);
  assert.equal(ctx.window.ACTIVE_SVG_BY_ASSET['dungeon/t-floor'], '<svg><rect/></svg>');
  assert.equal(ctx.window.ACTIVE_SVG_BY_ASSET['dungeon/not-string'], undefined);
});

test('applyActiveTileSetBundle normalizes svgByAsset keys with extensions', async () => {
  const ctx = await loadBrowserScript('static/js/map-tiles.js', { window: {} });
  const { applyActiveTileSetBundle } = exportSymbols(ctx, ['applyActiveTileSetBundle']);

  const ok = applyActiveTileSetBundle({
    tiles: [
      {
        id: 'd-floor',
        type: 'dungeon',
        edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
        weight: 1,
        asset: 'd-floor.png',
      },
    ],
    svgByAsset: {
      'dungeon/d-floor.png': '<svg><rect/></svg>',
    },
  });

  assert.equal(ok, true);
  assert.equal(ctx.window.ACTIVE_SVG_BY_ASSET['dungeon/d-floor'], '<svg><rect/></svg>');
  assert.equal(ctx.window.ACTIVE_SVG_BY_ASSET['dungeon/d-floor.png'], undefined);
});

test('loadActiveTileSetFromStorage returns false for invalid JSON', async () => {
  const storage = {
    getItem: () => '{ bad json',
    setItem: () => {},
  };
  const ctx = await loadBrowserScript('static/js/map-tiles.js', { window: { localStorage: storage } });
  const { loadActiveTileSetFromStorage } = exportSymbols(ctx, ['loadActiveTileSetFromStorage']);

  assert.equal(loadActiveTileSetFromStorage(), false);
});

import test from 'node:test';
import assert from 'node:assert/strict';
import { exportSymbols, loadBrowserScript } from './helpers/load-script.mjs';

test('splitTileAsset parses explicit extensions and extensionless assets', async () => {
  const ctx = await loadBrowserScript('static/js/map-engine.js');
  const { splitTileAsset } = exportSymbols(ctx, ['splitTileAsset']);

  const svg = splitTileAsset('d-floor.svg');
  const png = splitTileAsset('d-floor.png');
  const raw = splitTileAsset('d-floor');

  assert.equal(svg.base, 'd-floor');
  assert.equal(svg.ext, 'svg');
  assert.equal(png.base, 'd-floor');
  assert.equal(png.ext, 'png');
  assert.equal(raw.base, 'd-floor');
  assert.equal(raw.ext, '');
});

test('tileUrlFor encodes path segments', async () => {
  const ctx = await loadBrowserScript('static/js/map-engine.js');
  const { tileUrlFor } = exportSymbols(ctx, ['tileUrlFor']);

  const out = tileUrlFor('dungeon special', 'floor 1.svg');
  assert.equal(out, '/tiles/dungeon%20special/floor%201.svg');
});

test('getTileSVG uses svg-first fallback and then png', async () => {
  const calls = [];
  const fakeFetch = async (url) => {
    calls.push(url);
    if (url.endsWith('/d-floor.svg')) {
      return { ok: false, text: async () => '' };
    }
    if (url.endsWith('/d-floor.png')) {
      return { ok: true, text: async () => '' };
    }
    return { ok: false, text: async () => '' };
  };

  const ctx = await loadBrowserScript('static/js/map-engine.js', { fetch: fakeFetch, TILE: 40 });
  const { getTileSVG } = exportSymbols(ctx, ['getTileSVG']);

  const markup = await getTileSVG({ type: 'dungeon', asset: 'd-floor' });

  assert.ok(markup.includes('<image href="/tiles/dungeon/d-floor.png"'));
  assert.deepEqual(calls, ['/tiles/dungeon/d-floor.svg', '/tiles/dungeon/d-floor.png']);
});

test('getTileSVG ignores png-image override for non-png assets', async () => {
  const calls = [];
  const fakeFetch = async (url) => {
    calls.push(url);
    if (url.endsWith('/d-floor.svg')) {
      return { ok: true, text: async () => '<svg><rect id="floor"/></svg>' };
    }
    return { ok: false, text: async () => '' };
  };

  const ctx = await loadBrowserScript('static/js/map-engine.js', {
    fetch: fakeFetch,
    TILE: 40,
    window: {
      ACTIVE_SVG_BY_ASSET: {
        'dungeon/d-floor': '<svg><image href="/tiles/dungeon/d-floor.png" width="40" height="40"/></svg>',
      },
    },
  });
  const { getTileSVG } = exportSymbols(ctx, ['getTileSVG']);

  const markup = await getTileSVG({ type: 'dungeon', asset: 'd-floor' });
  assert.ok(markup.includes('id="floor"'));
  assert.deepEqual(calls, ['/tiles/dungeon/d-floor.svg']);
});

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import vm from 'node:vm';

function makeMathWithSequence(sequence = []) {
  const m = Object.create(Math);
  let i = 0;
  m.random = () => {
    if (i < sequence.length) {
      const v = sequence[i];
      i += 1;
      return v;
    }
    return 0.5;
  };
  return m;
}

export async function loadBrowserScript(relativePath, options = {}) {
  const absPath = path.join(process.cwd(), relativePath);
  const source = await readFile(absPath, 'utf8');

  const context = {
    console,
    setTimeout,
    clearTimeout,
    Promise,
    Map,
    Set,
    JSON,
    Date,
    String,
    Number,
    Boolean,
    Array,
    Object,
    RegExp,
    Math: makeMathWithSequence(options.randomSequence || []),
    fetch: options.fetch || (async () => ({ ok: false, text: async () => '' })),
    GRID: options.GRID ?? 20,
    TILE: options.TILE ?? 40,
    window: options.window || {},
    document: options.document || {
      getElementById: () => null,
      querySelectorAll: () => [],
    },
  };

  context.globalThis = context;
  vm.createContext(context);
  vm.runInContext(source, context, { filename: relativePath });

  return context;
}

export function exportSymbols(context, names) {
  const expr = '({ ' + names.join(', ') + ' })';
  return vm.runInContext(expr, context);
}

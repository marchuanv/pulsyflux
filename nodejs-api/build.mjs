import { build } from './node_modules/zig-build/src/index.ts';

await build({
  'broker_addon': {
    target: "x86_64-windows",
    output: 'broker_addon.node',
    sources: ['addon.cc'],
    napiVersion: 8
  }
}, {});

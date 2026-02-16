import { build } from './node_modules/zig-build/src/index.ts';

await build({
  'broker_addon': {
    output: 'build/Release/broker_addon.node',
    sources: ['addon.cc', 'broker_lib.lib'],
    napiVersion: 8
  }
}, {});

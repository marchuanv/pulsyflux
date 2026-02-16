import { build } from 'zig-build';

await build({
  'broker_addon': {
    output: 'build/Release/broker_addon.node',
    sources: ['addon.cc'],
    libraries: ['broker_lib'],
    librariesSearch: ['.'],
    napiVersion: 8
  }
});

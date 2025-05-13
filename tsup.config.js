import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['index.ts'],
  treeshake: true,
  format: 'esm',
  dts: true,
  clean: true,
});

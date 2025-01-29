import { build } from 'esbuild';

build({
    entryPoints: ['src/index.js'],
    bundle: true,
    minify: true,
    sourcemap: true,
    outfile: 'dist/bundle.js',
}).catch(() => process.exit(1))
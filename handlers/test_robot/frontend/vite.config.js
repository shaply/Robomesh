import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [
    svelte({
      compilerOptions: {
        customElement: false,
        // Inject styles at runtime so the plugin doesn't need a separate CSS file
        css: 'injected',
      },
    }),
  ],
  build: {
    lib: {
      entry: {
        robot_card: './src/RobotCard.svelte',
        robot_handler: './src/RobotHandler.svelte',
      },
      formats: ['es'],
    },
    outDir: '../dist',
    emptyOutDir: true,
    rollupOptions: {
      external: ['svelte', 'svelte/internal'],
    },
  },
});

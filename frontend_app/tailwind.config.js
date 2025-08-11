/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{html,js,ts}',
    './src/**/*.svelte'
  ],
  theme: {
    extend: {}
  },
  plugins: [],
  // Exclude processing of style blocks in Svelte files
  safelist: [],
  blocklist: []
}

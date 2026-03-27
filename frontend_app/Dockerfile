FROM node:22-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

# Install adapter-node for production builds
RUN npm i -D @sveltejs/adapter-node

COPY . .

# Override svelte.config.js to use adapter-node for production
RUN cat > svelte.config.js <<'EOF'
import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({ out: 'build' })
	}
};

export default config;
EOF

RUN rm -rf .svelte-kit build && npm run build

# --- Runtime ---
FROM node:22-alpine
WORKDIR /app

COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json .
COPY --from=builder /app/node_modules ./node_modules

ENV PORT=3000
EXPOSE 3000

CMD ["node", "build"]

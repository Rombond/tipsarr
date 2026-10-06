import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	const seerrProxy = env.VITE_SEERR_URL
		? {
				'/seerr-api': {
					target: env.VITE_SEERR_URL,
					changeOrigin: true,
					cookieDomainRewrite: '',
					rewrite: (path: string) => path.replace(/^\/seerr-api/, '')
				}
			}
		: undefined;

	return {
		plugins: [
			tailwindcss(),
			sveltekit({
				compilerOptions: {
					// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
					runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},

				// This is a client-only SPA (ssr = false everywhere, see src/routes/+layout.ts) —
				// adapter-static with a fallback builds a static bundle served behind any web
				// server (see deployments/), with SvelteKit's client router handling all routes.
				adapter: adapter({
					pages: 'build',
					assets: 'build',
					fallback: 'index.html',
					precompress: false,
					strict: true
				})
			})
		],
		server: {
			allowedHosts: ['server.brebond'],
			proxy: seerrProxy
		},
		preview: {
			proxy: seerrProxy
		}
	};
});

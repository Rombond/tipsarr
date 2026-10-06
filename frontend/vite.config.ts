import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	// Dev: the SPA and the Go backend look like one origin, so the session cookie stays first-party.
	const backend = env.TIPSARR_BACKEND_URL || 'http://localhost:8080';
	const proxy = { '/api': { target: backend, changeOrigin: false } };

	return {
		plugins: [
			tailwindcss(),
			sveltekit({
				compilerOptions: {
					// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
					runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},

				// Client-only SPA (ssr = false, see src/routes/+layout.ts). The Go backend embeds the
				// built `build/` folder and serves it, falling back to index.html for client routes.
				adapter: adapter({
					pages: 'build',
					assets: 'build',
					fallback: 'index.html',
					precompress: false,
					strict: true
				})
			})
		],
		server: { proxy },
		preview: { proxy }
	};
});

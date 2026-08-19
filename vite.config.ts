import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-auto';
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

				// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
				// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
				// See https://svelte.dev/docs/kit/adapters for more information about adapters.
				adapter: adapter()
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

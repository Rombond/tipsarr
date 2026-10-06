<script lang="ts">
	import { api, unwrap, expectOk } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';

	let { type, tmdbId }: { type: 'movie' | 'tv'; tmdbId: number } = $props();

	let watchlisted = $state(false);
	let blocklisted = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		const t = type;
		const id = tmdbId;
		unwrap(api.GET('/media/{type}/{id}/flags', { params: { path: { type: t, id } } }))
			.then((f) => {
				watchlisted = f.watchlisted;
				blocklisted = f.blocklisted;
			})
			.catch(() => {});
	});

	async function toggle(kind: 'watchlist' | 'blocklist') {
		busy = true;
		error = null;
		const on = kind === 'watchlist' ? watchlisted : blocklisted;
		try {
			if (kind === 'watchlist') {
				if (on) await expectOk(api.DELETE('/watchlist/{type}/{id}', { params: { path: { type, id: tmdbId } } }));
				else await expectOk(api.POST('/watchlist', { body: { type, tmdbId } }));
				watchlisted = !on;
			} else {
				if (on) await expectOk(api.DELETE('/blocklist/{type}/{id}', { params: { path: { type, id: tmdbId } } }));
				else await expectOk(api.POST('/blocklist', { body: { type, tmdbId } }));
				blocklisted = !on;
			}
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
</script>

<div class="flex flex-wrap items-center gap-2">
	<Button variant="outline" size="sm" disabled={busy} onclick={() => toggle('watchlist')}>
		{watchlisted ? '✓ On my watchlist' : '+ Watchlist'}
	</Button>
	<Button variant="ghost" size="sm" disabled={busy} onclick={() => toggle('blocklist')}>
		{blocklisted ? 'Hidden from my suggestions (undo)' : 'Hide from my suggestions'}
	</Button>
	{#if error}<span class="text-xs text-destructive">{error}</span>{/if}
</div>

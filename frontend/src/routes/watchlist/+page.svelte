<script lang="ts">
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let items = $state<Schemas['Item'][]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		unwrap(api.GET('/watchlist'))
			.then((r) => (items = r))
			.catch((e) => (error = (e as Error).message))
			.finally(() => (loading = false));
	});

	async function remove(item: Schemas['Item']) {
		const before = items;
		items = items.filter((i) => !(i.type === item.type && i.tmdbId === item.tmdbId));
		try {
			await expectOk(api.DELETE('/watchlist/{type}/{id}', { params: { path: { type: item.type, id: item.tmdbId } } }));
		} catch {
			items = before;
		}
	}
</script>

<svelte:head>
	<title>Watchlist · Tipsarr</title>
</svelte:head>

<div class="grid gap-4">
	<h1 class="font-bold text-2xl">Watchlist</h1>
	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading}
		<Skeleton class="h-56 w-full" />
	{:else if items.length === 0}
		<p class="text-sm text-muted-foreground">Nothing here yet. Open a movie or show and press "+ Watchlist".</p>
	{:else}
		<div class="flex flex-wrap gap-4">
			{#each items as item (`${item.type}:${item.tmdbId}`)}
				<MediaCard {item} onDismiss={remove} />
			{/each}
		</div>
		<p class="text-xs text-muted-foreground">Hover a poster and press ✕ to remove it from the list.</p>
	{/if}
</div>

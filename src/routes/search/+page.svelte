<script lang="ts">
	import { page } from '$app/state';
	import { search } from '$lib/api/seerr';
	import { fromSeerrResult, type MediaItem } from '$lib/api/media';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import PersonCard from '$lib/components/media/person-card.svelte';
	import MediaDetailModal from '$lib/components/media/media-detail-modal.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let query = $derived(page.url.searchParams.get('q') || '');

	let loading = $state(true);
	let error: Error | null = $state(null);
	let movies: MediaItem[] = $state([]);
	let tv: MediaItem[] = $state([]);
	let people: any[] = $state([]);

	let selected: MediaItem | null = $state(null);

	async function load() {
		if (!query) {
			movies = [];
			tv = [];
			people = [];
			loading = false;
			return;
		}
		loading = true;
		error = null;
		try {
			const data = await search(query);
			const results = data.results || [];
			movies = results.filter((r: any) => r.mediaType === 'movie').map(fromSeerrResult);
			tv = results.filter((r: any) => r.mediaType === 'tv').map(fromSeerrResult);
			people = results.filter((r: any) => r.mediaType === 'person');
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (query) load();
	});
</script>

<svelte:head>
	<meta name="description" content="TipsArr search" />
</svelte:head>

<div class="grid gap-8">
	<h1 class="font-bold text-2xl">Search results for "{query}"</h1>

	{#if loading}
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-6">
			{#each { length: 12 } as _, i (i)}
				<Skeleton class="aspect-[2/3] w-full rounded-lg" />
			{/each}
		</div>
	{:else if error}
		<div class="text-sm text-destructive">
			Error: {error.message}
			<button class="ml-2 underline" onclick={load}>Retry</button>
		</div>
	{:else if !movies.length && !tv.length && !people.length}
		<p class="text-muted-foreground text-sm">No results found.</p>
	{:else}
		{#if movies.length}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">Movies</h2>
				<div class="flex flex-wrap gap-4">
					{#each movies as item (item.key)}
						<MediaCard {item} onSelect={(i) => (selected = i)} />
					{/each}
				</div>
			</section>
		{/if}
		{#if tv.length}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">TV Shows</h2>
				<div class="flex flex-wrap gap-4">
					{#each tv as item (item.key)}
						<MediaCard {item} onSelect={(i) => (selected = i)} />
					{/each}
				</div>
			</section>
		{/if}
		{#if people.length}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">People</h2>
				<div class="flex flex-wrap gap-4">
					{#each people as person (person.id)}
						<PersonCard {person} />
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</div>

<MediaDetailModal open={selected !== null} item={selected} onclose={() => (selected = null)} />

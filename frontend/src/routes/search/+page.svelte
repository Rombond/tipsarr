<script lang="ts">
	import { page } from '$app/state';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import PersonCard from '$lib/components/media/person-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let query = $derived(page.url.searchParams.get('q') || '');

	let loading = $state(true);
	let error = $state<Error | null>(null);
	let result = $state<Schemas['SearchResult'] | null>(null);

	const movies = $derived(result?.items.filter((i) => i.type === 'movie') ?? []);
	const shows = $derived(result?.items.filter((i) => i.type === 'tv') ?? []);
	const people = $derived(result?.people ?? []);

	async function load() {
		if (!query) {
			result = null;
			loading = false;
			return;
		}
		loading = true;
		error = null;
		try {
			result = await unwrap(api.GET('/search', { params: { query: { q: query } } }));
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		query;
		load();
	});
</script>

<svelte:head>
	<title>Search · Tipsarr</title>
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
	{:else if !movies.length && !shows.length && !people.length}
		<p class="text-muted-foreground text-sm">No results found.</p>
	{:else}
		{#if movies.length}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">Movies</h2>
				<div class="flex flex-wrap gap-4">
					{#each movies as item (item.tmdbId)}
						<MediaCard {item} />
					{/each}
				</div>
			</section>
		{/if}
		{#if shows.length}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">TV Shows</h2>
				<div class="flex flex-wrap gap-4">
					{#each shows as item (item.tmdbId)}
						<MediaCard {item} />
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

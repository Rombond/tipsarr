<script lang="ts">
	import { page } from '$app/state';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import PersonCard from '$lib/components/media/person-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let query = $derived(page.url.searchParams.get('q') || '');

	type Tab = 'all' | 'movies' | 'tv' | 'people';
	let tab = $state<Tab>('all');
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
		tab = 'all';
		load();
	});
</script>

<svelte:head>
	<title>Search · Tipsarr</title>
</svelte:head>

<div class="grid gap-8">
	<div>
		<h1 class="font-bold text-3xl">Results for “{query}”</h1>
		{#if result && !loading}
			<p class="text-sm text-muted-foreground">{movies.length} movies · {shows.length} TV shows · {people.length} people</p>
		{/if}
	</div>

	{#if result && !loading && movies.length + shows.length + people.length > 0}
		<div class="flex gap-1 overflow-x-auto" role="tablist" aria-label="Result type">
			{#each [['all', 'All', movies.length + shows.length + people.length], ['movies', 'Movies', movies.length], ['tv', 'TV shows', shows.length], ['people', 'People', people.length]] as [id, label, n] (id)}
				<button
					type="button"
					role="tab"
					aria-selected={tab === id}
					class="shrink-0 cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors {tab === id ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
					onclick={() => (tab = id as Tab)}
				>
					{label} <span class="text-[0.85em]">({n})</span>
				</button>
			{/each}
		</div>
	{/if}

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
		<div class="rounded-xl border border-dashed border-border p-8 text-center">
			<p class="font-medium">Nothing found for “{query}”</p>
			<p class="mt-1 text-sm text-muted-foreground">Check the spelling, or try the original title or fewer words.</p>
		</div>
	{:else}
		{#if movies.length && (tab === 'all' || tab === 'movies')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">Movies</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each movies as item (item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
		{#if shows.length && (tab === 'all' || tab === 'tv')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">TV Shows</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each shows as item (item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
		{#if people.length && (tab === 'all' || tab === 'people')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">People</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each people as person (person.id)}
						<PersonCard {person} fluid />
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</div>

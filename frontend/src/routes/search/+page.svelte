<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
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
	const keywords = $derived(result?.keywords ?? []);
	const tagged = $derived(result?.tagged ?? []);

	async function load() {
		if (!query) {
			result = null;
			loading = false;
			return;
		}
		loading = true;
		error = null;
		try {
			result = await unwrap(api.GET('/search', { params: { query: { q: query, tags: true } } }));
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
	<title>{t('search.page_title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-8">
	<div>
		<h1 class="font-bold text-3xl">{t('search.results_for', { query })}</h1>
		{#if result && !loading}
			<p class="text-sm text-muted-foreground">{t('search.counts', { movies: movies.length, shows: shows.length, people: people.length })}</p>
		{/if}
	</div>

	{#if keywords.length && !loading}
		<div class="flex flex-wrap items-center gap-1.5">
			<span class="text-sm text-muted-foreground">{t('search.tags')}</span>
			{#each keywords as k (k.id)}
				<a href="/browse?keyword={k.id}&name={encodeURIComponent(k.name)}" class="rounded-md bg-muted px-2 py-1 text-xs hover:bg-primary hover:text-primary-foreground">{k.name}</a>
			{/each}
		</div>
	{/if}

	{#if result && !loading && movies.length + shows.length + people.length > 0}
		<div class="flex gap-1 overflow-x-auto" role="tablist" aria-label={t('search.page_title')}>
			{#each [['all', t('search.tab_all'), movies.length + shows.length + people.length], ['movies', t('type.movies'), movies.length], ['tv', t('type.shows'), shows.length], ['people', t('type.people'), people.length]] as [id, label, n] (id)}
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
			{t('common.error_prefix', { message: errorText(error) })}
			<button class="ml-2 underline" onclick={load}>{t('common.retry')}</button>
		</div>
	{:else if !movies.length && !shows.length && !people.length && !tagged.length}
		<div class="rounded-xl border border-dashed border-border p-8 text-center">
			<p class="font-medium">{t('search.nothing', { query })}</p>
			<p class="mt-1 text-sm text-muted-foreground">{t('search.nothing_hint')}</p>
		</div>
	{:else}
		{#if movies.length && (tab === 'all' || tab === 'movies')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">{t('type.movies')}</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each movies as item (item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
		{#if shows.length && (tab === 'all' || tab === 'tv')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">{t('type.shows')}</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each shows as item (item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
		{#if tagged.length && tab === 'all'}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">{t('search.tagged', { tags: keywords.map((k) => k.name).join(', ') })}</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each tagged as item (item.type + item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
		{#if people.length && (tab === 'all' || tab === 'people')}
			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">{t('type.people')}</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each people as person (person.id)}
						<PersonCard {person} fluid />
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</div>

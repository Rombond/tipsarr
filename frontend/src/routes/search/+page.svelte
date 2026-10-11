<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import PersonCard from '$lib/components/media/person-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { Input } from '$lib/components/ui/input';
	import FilterIcon from '@lucide/svelte/icons/sliders-horizontal';
	import XIcon from '@lucide/svelte/icons/x';

	let query = $derived(page.url.searchParams.get('q') || '');

	type Tab = 'all' | 'movies' | 'tv' | 'people';
	let tab = $state<Tab>('all');
	let loading = $state(true);
	let error = $state<Error | null>(null);
	let result = $state<Schemas['SearchResult'] | null>(null);

	// Filters apply to the titles already loaded (people are not filtered).
	type Sort = 'relevance' | 'rating' | 'year';
	let panelOpen = $state(false);
	let yearFrom = $state('');
	let yearTo = $state('');
	let minRating = $state('');
	let avail = $state('any');
	let sort = $state<Sort>('relevance');
	const activeFilters = $derived([yearFrom, yearTo, minRating].filter(Boolean).length + (avail !== 'any' ? 1 : 0) + (sort !== 'relevance' ? 1 : 0));

	function yearOf(i: Schemas['Item']) {
		return Number(i.releaseDate?.slice(0, 4)) || 0;
	}
	function refine(items: Schemas['Item'][]) {
		const from = Number(yearFrom) || 0;
		const to = Number(yearTo) || 0;
		const min = Number(minRating) || 0;
		const out = items.filter((i) => {
			if (from && yearOf(i) < from) return false;
			if (to && (!yearOf(i) || yearOf(i) > to)) return false;
			if (min && i.voteAverage < min) return false;
			if (avail === 'in' && i.availability !== 'available') return false;
			if (avail === 'missing' && i.availability === 'available') return false;
			return true;
		});
		if (sort === 'rating') out.sort((a, b) => b.voteAverage - a.voteAverage);
		if (sort === 'year') out.sort((a, b) => yearOf(b) - yearOf(a));
		return out;
	}
	function resetFilters() {
		yearFrom = yearTo = minRating = '';
		avail = 'any';
		sort = 'relevance';
	}

	const allMovies = $derived(result?.items.filter((i) => i.type === 'movie') ?? []);
	const allShows = $derived(result?.items.filter((i) => i.type === 'tv') ?? []);
	const movies = $derived(refine(allMovies));
	const shows = $derived(refine(allShows));
	const people = $derived(result?.people ?? []);
	const keywords = $derived(result?.keywords ?? []);
	const tagged = $derived(refine(result?.tagged ?? []));
	const filtering = $derived(activeFilters > 0);

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

	const sortOptions = $derived([
		{ value: 'relevance', label: t('search.sort_relevance') },
		{ value: 'rating', label: t('library.sort_rating') },
		{ value: 'year', label: t('library.sort_year') },
	]);
	const ratingOptions = $derived([{ value: '', label: t('library.any') }, ...[5, 6, 7, 8].map((n) => ({ value: String(n), label: `${n}+` }))]);
	const availOptions = $derived([
		{ value: 'any', label: t('library.any') },
		{ value: 'missing', label: t('search.avail_missing') },
		{ value: 'in', label: t('search.avail_in') },
	]);

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

	{#if result && !loading && allMovies.length + allShows.length > 0}
		<div class="grid gap-3">
			<div class="flex flex-wrap items-center gap-2">
				<button
					type="button"
					aria-expanded={panelOpen}
					class="flex cursor-pointer items-center gap-1.5 rounded-full border px-3 py-1 text-sm transition-colors {activeFilters ? 'border-primary text-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
					onclick={() => (panelOpen = !panelOpen)}
				>
					<FilterIcon class="size-3.5" />{t('search.filters')}{#if activeFilters} ({activeFilters}){/if}
				</button>
				{#if activeFilters}
					<button type="button" class="flex cursor-pointer items-center gap-1 text-sm text-muted-foreground underline-offset-4 hover:underline" onclick={resetFilters}><XIcon class="size-3.5" />{t('search.filters_reset')}</button>
				{/if}
			</div>
			{#if panelOpen}
				<div class="grid gap-3 rounded-lg border border-border p-4 sm:grid-cols-2 lg:grid-cols-4">
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.year')}</span>
						<div class="flex items-center gap-2">
							<Input bind:value={yearFrom} inputmode="numeric" maxlength={4} aria-label={t('library.year_from')} />
							<span class="text-muted-foreground">–</span>
							<Input bind:value={yearTo} inputmode="numeric" maxlength={4} aria-label={t('library.year_to')} />
						</div>
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.min_rating')}</span>
						<SimpleSelect label={t('library.min_rating')} value={minRating} options={ratingOptions} onchange={(v) => (minRating = v)} class="w-full" />
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('search.avail')}</span>
						<SimpleSelect label={t('search.avail')} value={avail} options={availOptions} onchange={(v) => (avail = v)} class="w-full" />
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.sort_by')}</span>
						<SimpleSelect label={t('library.sort_by')} value={sort} options={sortOptions} onchange={(v) => (sort = v as Sort)} class="w-full" />
					</div>
				</div>
			{/if}
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
			{#if filtering}
				<p class="font-medium">{t('search.no_match')}</p>
			{:else}
				<p class="font-medium">{t('search.nothing', { query })}</p>
				<p class="mt-1 text-sm text-muted-foreground">{t('search.nothing_hint')}</p>
			{/if}
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

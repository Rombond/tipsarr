<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import { asMediaItem } from '$lib/library-item';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import SearchIcon from '@lucide/svelte/icons/search';
	import FilterIcon from '@lucide/svelte/icons/sliders-horizontal';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import XIcon from '@lucide/svelte/icons/x';

	type Sort = 'added' | 'title' | 'year' | 'rating' | 'runtime' | 'popular';
	const SORTS: Sort[] = ['added', 'title', 'year', 'rating', 'runtime', 'popular'];
	const PAGE_SIZE = 48;
	const GENRES_SHOWN = 14;

	// The filters live in the address, so a refresh, the back button and a shared link keep them.
	const q0 = page.url.searchParams;
	let text = $state(q0.get('q') ?? '');
	let type = $state<'all' | 'movie' | 'tv'>((['movie', 'tv'].includes(q0.get('type') ?? '') ? q0.get('type') : 'all') as 'all' | 'movie' | 'tv');
	let genres = $state<string[]>(q0.getAll('genre'));
	let yearFrom = $state(q0.get('yearFrom') ?? '');
	let yearTo = $state(q0.get('yearTo') ?? '');
	let minRating = $state(q0.get('minRating') ?? '');
	let maxRuntime = $state(q0.get('maxRuntime') ?? '');
	let watched = $state(q0.get('watched') ?? 'any');
	let sort = $state<Sort>(SORTS.includes(q0.get('sort') as Sort) ? (q0.get('sort') as Sort) : 'added');
	let desc = $state(q0.get('dir') !== 'asc');
	let panelOpen = $state(false);
	let allGenres = $state(false);

	let facets = $state<Schemas['LibraryFacets'] | null>(null);
	let items = $state<Schemas['LibraryItem'][]>([]);
	let total = $state(0);
	let pageNo = $state(0);
	let totalPages = $state(1);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let sentinel = $state<HTMLElement | undefined>();

	// what the server is asked, as one string: a change starts a fresh list
	const query = $derived.by(() => {
		const p = new URLSearchParams();
		if (text.trim()) p.set('q', text.trim());
		if (type !== 'all') p.set('type', type);
		for (const g of genres) p.append('genre', g);
		if (yearFrom) p.set('yearFrom', yearFrom);
		if (yearTo) p.set('yearTo', yearTo);
		if (minRating) p.set('minRating', minRating);
		if (maxRuntime) p.set('maxRuntime', maxRuntime);
		if (watched !== 'any') p.set('watched', watched);
		if (sort !== 'added') p.set('sort', sort);
		if (!desc) p.set('dir', 'asc');
		return p.toString();
	});
	const activeFilters = $derived(genres.length + [yearFrom, yearTo, minRating, maxRuntime].filter(Boolean).length + (watched !== 'any' ? 1 : 0));
	const shownGenres = $derived(allGenres ? (facets?.genres ?? []) : (facets?.genres ?? []).slice(0, GENRES_SHOWN));

	let typingTimer: ReturnType<typeof setTimeout>;
	let debounced = $state(untrack(() => query));
	$effect(() => {
		const v = query;
		clearTimeout(typingTimer);
		typingTimer = setTimeout(() => (debounced = v), 250);
		return () => clearTimeout(typingTimer);
	});

	async function loadNext() {
		if (loading || pageNo >= totalPages) return;
		loading = true;
		error = null;
		const mine = debounced;
		try {
			const res = await unwrap(
				api.GET('/library', {
					params: {
						query: {
							type,
							q: text.trim() || undefined,
							genre: genres.length ? genres : undefined,
							yearFrom: Number(yearFrom) || undefined,
							yearTo: Number(yearTo) || undefined,
							minRating: Number(minRating) || undefined,
							maxRuntime: Number(maxRuntime) || undefined,
							watched: watched as 'any' | 'yes' | 'no',
							sort,
							dir: desc ? 'desc' : 'asc',
							page: pageNo + 1,
							pageSize: PAGE_SIZE,
						},
					},
				}),
			);
			if (mine !== debounced) return; // the filters changed meanwhile
			items = [...items, ...res.items];
			total = res.total;
			pageNo = res.page;
			totalPages = Math.max(1, res.totalPages);
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	// new filters -> fresh list, and the address follows
	$effect(() => {
		const key = debounced;
		untrack(() => {
			items = [];
			pageNo = 0;
			totalPages = 1;
			total = 0;
			loading = false;
			goto(key ? `?${key}` : page.url.pathname, { replaceState: true, keepFocus: true, noScroll: true });
			loadNext();
		});
	});

	// Scrolling near the end loads the next page. The observer is rebuilt after every page: it
	// reports a change only, so on a big screen where the first pages do not fill the view it would
	// otherwise never ask again (a fresh observer reports the current state straight away).
	$effect(() => {
		items.length;
		pageNo;
		if (!sentinel || loading) return;
		const io = new IntersectionObserver((e) => e[0].isIntersecting && !error && untrack(loadNext), { rootMargin: '600px' });
		io.observe(sentinel);
		return () => io.disconnect();
	});

	onMount(() => {
		unwrap(api.GET('/library/facets'))
			.then((f) => (facets = f))
			.catch(() => {});
		if (activeFilters) panelOpen = true;
	});

	function toggleGenre(g: string) {
		genres = genres.includes(g) ? genres.filter((x) => x !== g) : [...genres, g];
	}

	function reset() {
		text = '';
		type = 'all';
		genres = [];
		yearFrom = yearTo = minRating = maxRuntime = '';
		watched = 'any';
		sort = 'added';
		desc = true;
	}

	const sortOptions = $derived(SORTS.map((s) => ({ value: s, label: t(`library.sort_${s}` as 'library.sort_added') })));
	const ratingOptions = $derived([
		{ value: '', label: t('library.any') },
		...[5, 6, 7, 8].map((n) => ({ value: String(n), label: `${n}+` })),
	]);
	const runtimeOptions = $derived([
		{ value: '', label: t('library.any') },
		...[90, 120, 150].map((n) => ({ value: String(n), label: t('library.under', { minutes: n }) })),
	]);
	const watchedOptions = $derived([
		{ value: 'any', label: t('library.any') },
		{ value: 'no', label: t('library.not_watched') },
		{ value: 'yes', label: t('library.watched') },
	]);
	const typeTabs = $derived([
		{ id: 'all', label: t('library.all'), n: (facets?.movies ?? 0) + (facets?.shows ?? 0) },
		{ id: 'movie', label: t('type.movies'), n: facets?.movies ?? 0 },
		{ id: 'tv', label: t('type.shows'), n: facets?.shows ?? 0 },
	] as const);
</script>

<svelte:head>
	<title>{t('library.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-5">
	<div class="grid gap-1">
		<h1 class="text-3xl font-bold">{t('library.title')}</h1>
		<p class="text-sm text-muted-foreground">{facets ? t('library.subtitle', { movies: facets.movies, shows: facets.shows }) : ' '}</p>
	</div>

	<div class="grid gap-3">
		<div class="flex flex-wrap items-center gap-2">
			<div class="relative min-w-48 flex-1">
				<SearchIcon class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
				<Input bind:value={text} class="pl-9" placeholder={t('library.search')} aria-label={t('library.search')} />
			</div>
			<SimpleSelect label={t('library.sort_by')} value={sort} options={sortOptions} onchange={(v) => (sort = v as Sort)} class="w-44" />
			<Button variant="outline" size="icon" aria-label={desc ? t('library.descending') : t('library.ascending')} title={desc ? t('library.descending') : t('library.ascending')} onclick={() => (desc = !desc)}>
				{#if desc}<ArrowDownIcon />{:else}<ArrowUpIcon />{/if}
			</Button>
			<Button variant={panelOpen ? 'secondary' : 'outline'} onclick={() => (panelOpen = !panelOpen)} aria-expanded={panelOpen}>
				<FilterIcon /> {t('library.filters')}{#if activeFilters}<span class="ml-1 rounded-full bg-primary px-1.5 text-xs text-primary-foreground">{activeFilters}</span>{/if}
			</Button>
		</div>

		<div class="flex flex-wrap items-center gap-1" role="tablist" aria-label={t('library.title')}>
			{#each typeTabs as tab (tab.id)}
				<button
					type="button"
					role="tab"
					aria-selected={type === tab.id}
					class="cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors {type === tab.id ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
					onclick={() => (type = tab.id)}
				>{tab.label}{#if facets}<span class="ml-1 opacity-70">{tab.n}</span>{/if}</button>
			{/each}
			{#if activeFilters || text || type !== 'all'}
				<button type="button" class="ml-1 flex cursor-pointer items-center gap-1 text-sm text-muted-foreground underline-offset-4 hover:underline" onclick={reset}><XIcon class="size-3.5" />{t('library.reset')}</button>
			{/if}
		</div>

		{#if panelOpen}
			<div class="grid gap-4 rounded-lg border border-border p-4">
				{#if facets?.genres.length}
					<div class="grid gap-2">
						<span class="text-sm font-medium">{t('library.genres')}</span>
						<div class="flex flex-wrap gap-1.5">
							{#each shownGenres as g (g.name)}
								<button
									type="button"
									aria-pressed={genres.includes(g.name)}
									class="cursor-pointer rounded-full border px-2.5 py-0.5 text-xs transition-colors {genres.includes(g.name) ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
									onclick={() => toggleGenre(g.name)}
								>{g.name} <span class="opacity-70">{g.count}</span></button>
							{/each}
							{#if facets.genres.length > GENRES_SHOWN}
								<button type="button" class="cursor-pointer px-1 text-xs text-muted-foreground underline-offset-4 hover:underline" onclick={() => (allGenres = !allGenres)}>
									{allGenres ? t('library.fewer') : t('library.more_genres', { count: facets.genres.length - GENRES_SHOWN })}
								</button>
							{/if}
						</div>
						{#if genres.length > 1}<p class="text-xs text-muted-foreground">{t('library.genres_all')}</p>{/if}
					</div>
				{/if}
				<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.year')}</span>
						<div class="flex items-center gap-2">
							<Input bind:value={yearFrom} inputmode="numeric" maxlength={4} placeholder={String(facets?.yearMin || '')} aria-label={t('library.year_from')} />
							<span class="text-muted-foreground">–</span>
							<Input bind:value={yearTo} inputmode="numeric" maxlength={4} placeholder={String(facets?.yearMax || '')} aria-label={t('library.year_to')} />
						</div>
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.min_rating')}</span>
						<SimpleSelect label={t('library.min_rating')} value={minRating} options={ratingOptions} onchange={(v) => (minRating = v)} class="w-full" />
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.runtime')}</span>
						<SimpleSelect label={t('library.runtime')} value={maxRuntime} options={runtimeOptions} onchange={(v) => (maxRuntime = v)} class="w-full" />
					</div>
					<div class="grid gap-1 text-sm">
						<span class="font-medium">{t('library.watched_state')}</span>
						<SimpleSelect label={t('library.watched_state')} value={watched} options={watchedOptions} onchange={(v) => (watched = v)} class="w-full" />
					</div>
				</div>
			</div>
		{/if}
	</div>

	{#if error}
		<p class="text-sm text-destructive">{t('common.error_prefix', { message: error })} <button class="ml-2 underline" onclick={loadNext}>{t('common.retry')}</button></p>
	{:else if items.length === 0 && !loading}
		<p class="text-sm text-muted-foreground">{facets && facets.movies + facets.shows === 0 ? t('library.empty') : t('common.no_results')}</p>
	{:else}
		<p class="text-xs text-muted-foreground" aria-live="polite">{t('library.count', { count: total })}</p>
		<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
			{#each items as item (item.type + item.tmdbId)}
				<MediaCard item={asMediaItem(item)} posterUrl={item.posterUrl} hideStatus watched={item.watched} fluid />
			{/each}
			{#if loading}
				{#each { length: 8 } as _, i (i)}<Skeleton class="aspect-[2/3] w-full rounded-lg" />{/each}
			{/if}
		</div>
		<div bind:this={sentinel} class="h-px"></div>
	{/if}
</div>

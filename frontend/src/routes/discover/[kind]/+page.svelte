<script lang="ts">
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type MediaItem } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import MediaDetailModal from '$lib/components/media/media-detail-modal.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	// The full list behind a Discover row: /discover/trending, /discover/popular-movies,
	// /discover/popular-tv and /discover/upcoming, loaded page after page as you scroll.
	type Result = { page: number; totalPages: number; items: MediaItem[] };
	const kinds: Record<string, { title: string; hint: string; load: (page: number) => Promise<Result> }> = {
		trending: { title: 'discover.trending', hint: 'discover.trending_hint', load: (p) => unwrap(api.GET('/discover/trending', { params: { query: { page: p } } })) },
		'popular-movies': { title: 'discover.popular_movies', hint: 'discover.popular_hint', load: (p) => unwrap(api.GET('/discover/movies', { params: { query: { page: p } } })) },
		'popular-tv': { title: 'discover.popular_tv', hint: 'discover.popular_hint', load: (p) => unwrap(api.GET('/discover/tv', { params: { query: { page: p } } })) },
		upcoming: { title: 'discover.upcoming', hint: 'discover.upcoming_hint', load: (p) => unwrap(api.GET('/discover/upcoming', { params: { query: { page: p } } })) },
	};

	const kind = $derived(kinds[page.params.kind ?? ''] ?? null);

	let items = $state<MediaItem[]>([]);
	let pageNo = $state(0);
	let totalPages = $state(1);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let sentinel = $state<HTMLElement | undefined>();
	let selected = $state<MediaItem | null>(null);

	async function loadNext() {
		if (!kind || loading || pageNo >= totalPages) return;
		loading = true;
		error = null;
		const mine = page.params.kind;
		try {
			const res = await kind.load(pageNo + 1);
			if (mine !== page.params.kind) return; // the person went to another list meanwhile
			const seen = new Set(items.map((i) => `${i.type}:${i.tmdbId}`));
			items = [...items, ...res.items.filter((i) => !seen.has(`${i.type}:${i.tmdbId}`))];
			pageNo = res.page;
			totalPages = res.totalPages;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	// another list starts afresh
	$effect(() => {
		page.params.kind;
		untrack(() => {
			items = [];
			pageNo = 0;
			totalPages = 1;
			loading = false;
			loadNext();
		});
	});

	// scrolling near the end loads the next page; rebuilt after every page, as a fresh observer
	// reports the current state (on a big screen the first pages may not fill the view)
	$effect(() => {
		items.length;
		if (!sentinel || loading) return;
		const io = new IntersectionObserver((e) => e[0].isIntersecting && !error && untrack(loadNext), { rootMargin: '600px' });
		io.observe(sentinel);
		return () => io.disconnect();
	});
</script>

<svelte:head>
	<title>{kind ? t(kind.title as 'discover.trending') : t('discover.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-5">
	<div class="grid gap-1">
		<a href="/discover" class="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"><ArrowLeftIcon class="size-4" />{t('discover.title')}</a>
		<h1 class="text-3xl font-bold">{kind ? t(kind.title as 'discover.trending') : ''}</h1>
		{#if kind}<p class="max-w-2xl text-sm text-muted-foreground">{t(kind.hint as 'discover.trending_hint')}</p>{/if}
	</div>

	{#if !kind}
		<p class="text-sm text-muted-foreground">{t('common.no_results')}</p>
	{:else if error}
		<p class="text-sm text-destructive">{t('common.error_prefix', { message: error })} <button class="ml-2 underline" onclick={loadNext}>{t('common.retry')}</button></p>
	{:else if items.length === 0 && !loading}
		<p class="text-sm text-muted-foreground">{t('common.no_results')}</p>
	{:else}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
			{#each items as item (item.type + item.tmdbId)}
				<MediaCard {item} fluid onSelect={(i) => (selected = i)} />
			{/each}
			{#if loading}
				{#each { length: 8 } as _, i (i)}<Skeleton class="aspect-[2/3] w-full rounded-lg" />{/each}
			{/if}
		</div>
		<div bind:this={sentinel} class="h-px"></div>
		{#if error}<Button variant="outline" onclick={loadNext}>{t('common.retry')}</Button>{/if}
	{/if}
</div>

<MediaDetailModal open={selected !== null} item={selected} onclose={() => (selected = null)} />

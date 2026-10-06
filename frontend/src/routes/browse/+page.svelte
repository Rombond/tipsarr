<script lang="ts">
	import { untrack } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { api, unwrap, errorText, type MediaItem } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import TagIcon from '@lucide/svelte/icons/tag';

	// /browse?type=movie&genre=28  or  /browse?keyword=1234  (a tag spans movies and shows)
	const genre = $derived(Number(page.url.searchParams.get('genre')) || 0);
	const keyword = $derived(Number(page.url.searchParams.get('keyword')) || 0);
	const nameParam = $derived(page.url.searchParams.get('name') ?? '');
	const typeParam = $derived(page.url.searchParams.get('type') === 'tv' ? 'tv' : 'movie');

	let tab = $state<'movie' | 'tv'>('movie');
	let items = $state<MediaItem[]>([]);
	let pageNo = $state(0);
	let totalPages = $state(1);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let label = $state('');

	const mediaType = $derived(keyword ? tab : typeParam);

	async function loadNext() {
		if (loading || (!genre && !keyword)) return;
		loading = true;
		error = null;
		const mine = `${mediaType}:${genre}:${keyword}`;
		try {
			const path = mediaType === 'tv' ? '/discover/tv' : '/discover/movies';
			const res = await unwrap(api.GET(path, { params: { query: { page: pageNo + 1, genre, keyword } } }));
			if (mine !== `${mediaType}:${genre}:${keyword}`) return; // the person navigated elsewhere meanwhile
			const seen = new Set(items.map((i) => i.tmdbId));
			items = [...items, ...res.items.filter((i) => !seen.has(i.tmdbId))];
			pageNo = res.page;
			totalPages = res.totalPages;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	// a new genre/tag/type starts a fresh list
	$effect(() => {
		mediaType;
		genre;
		keyword;
		untrack(() => {
			items = [];
			pageNo = 0;
			totalPages = 1;
			loading = false;
			loadNext();
		});
	});

	// the page title: the name we were given, else look it up
	$effect(() => {
		label = nameParam;
		if (nameParam) return;
		if (keyword) unwrap(api.GET('/discover/keywords/{id}', { params: { path: { id: keyword } } })).then((k) => (label = k.name)).catch(() => {});
		else if (genre)
			unwrap(api.GET('/discover/genres/{type}', { params: { path: { type: typeParam } } }))
				.then((g) => (label = g.find((x) => x.id === genre)?.name ?? ''))
				.catch(() => {});
	});
</script>

<svelte:head>
	<title>{label || t('browse.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-6">
	<div class="grid gap-1">
		<p class="flex items-center gap-1.5 text-sm text-muted-foreground">
			{#if keyword}<TagIcon class="size-4" />{t('browse.tag')}{:else}{typeParam === 'tv' ? t('type.shows') : t('type.movies')}{/if}
		</p>
		<h1 class="font-bold text-3xl">{label || '…'}</h1>
	</div>

	{#if keyword}
		<div class="flex gap-1" role="tablist" aria-label={t('browse.title')}>
			{#each [['movie', t('type.movies')], ['tv', t('type.shows')]] as [id, name] (id)}
				<button
					type="button"
					role="tab"
					aria-selected={tab === id}
					class="cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors {tab === id ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
					onclick={() => (tab = id as 'movie' | 'tv')}
				>{name}</button>
			{/each}
		</div>
	{/if}

	{#if !genre && !keyword}
		<p class="text-sm text-muted-foreground">{t('browse.nothing_selected')}</p>
	{:else if error}
		<p class="text-sm text-destructive">{t('common.error_prefix', { message: error })} <button class="ml-2 underline" onclick={loadNext}>{t('common.retry')}</button></p>
	{:else if items.length === 0 && !loading}
		<p class="text-sm text-muted-foreground">{t('common.no_results')}</p>
	{:else}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
			{#each items as item (item.type + item.tmdbId)}
				<MediaCard {item} fluid />
			{/each}
			{#if loading}
				{#each { length: 6 } as _, i (i)}<Skeleton class="aspect-[2/3] w-full rounded-lg" />{/each}
			{/if}
		</div>
		{#if pageNo < totalPages && !loading}
			<div class="flex justify-center"><Button variant="outline" onclick={loadNext}>{t('browse.more')}</Button></div>
		{/if}
	{/if}
</div>

<script lang="ts">
	import { getTrending, getDiscoverMovies, getDiscoverTv } from '$lib/api/seerr';
	import { fromSeerrResult, type MediaItem } from '$lib/api/media';
	import Carousel from '$lib/components/media/carousel.svelte';

	let {
		title,
		source,
		onSelect,
	}: { title: string; source: 'trending' | 'movies' | 'tv'; onSelect?: (item: MediaItem) => void } = $props();

	function fetcher(p: number) {
		if (source === 'trending') return getTrending(p);
		if (source === 'movies') return getDiscoverMovies(p);
		return getDiscoverTv(p);
	}

	let items: MediaItem[] = $state([]);
	let loading = $state(true);
	let error: Error | null = $state(null);
	let page = 1;
	let hasMore = $state(true);

	async function loadPage(reset = false) {
		if (reset) {
			page = 1;
			items = [];
			hasMore = true;
		}
		loading = true;
		error = null;
		try {
			const data = await fetcher(page);
			const results = (data.results || []).filter((r: any) => r.mediaType !== 'person').map(fromSeerrResult);
			items = reset ? results : [...items, ...results];
			hasMore = page < (data.totalPages || 1);
			page += 1;
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	loadPage(true);
</script>

<Carousel {title} {items} {loading} {error} {hasMore} onLoadMore={() => loadPage(false)} {onSelect} onRetry={() => loadPage(true)} />

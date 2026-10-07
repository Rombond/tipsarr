<script lang="ts">
	import { untrack } from 'svelte';
	import type { MediaItem } from '$lib/api/client';
	import Carousel from '$lib/components/media/carousel.svelte';

	type Page = { page: number; totalPages: number; items: MediaItem[] };

	let {
		title,
		load,
		onSelect,
		href,
	}: { title: string; load: (page: number) => Promise<Page>; onSelect?: (item: MediaItem) => void; href?: string } = $props();

	let items: MediaItem[] = $state([]);
	let page = $state(0);
	let totalPages = $state(1);
	let loading = $state(false);
	let error: Error | null = $state(null);

	async function loadNext() {
		if (loading) return;
		loading = true;
		error = null;
		try {
			const res = await load(page + 1);
			const seen = new Set(items.map((i) => `${i.type}:${i.tmdbId}`));
			items = [...items, ...res.items.filter((i) => !seen.has(`${i.type}:${i.tmdbId}`))];
			page = res.page;
			totalPages = res.totalPages;
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	// first page only: loadNext reads/writes `loading`, which would re-trigger this effect and
	// page through the whole catalogue endlessly if it were tracked
	$effect(() => {
		untrack(loadNext);
	});
</script>

<Carousel {title} {href} {items} {loading} {error} hasMore={page < totalPages} onLoadMore={loadNext} {onSelect} onRetry={loadNext} />

<script lang="ts">
	import type { MediaItem } from '$lib/api/client';
	import Carousel from '$lib/components/media/carousel.svelte';

	type Page = { page: number; totalPages: number; items: MediaItem[] };

	let {
		title,
		load,
		onSelect,
	}: { title: string; load: (page: number) => Promise<Page>; onSelect?: (item: MediaItem) => void } = $props();

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

	$effect(() => {
		loadNext();
	});
</script>

<Carousel {title} {items} {loading} {error} hasMore={page < totalPages} onLoadMore={loadNext} {onSelect} onRetry={loadNext} />

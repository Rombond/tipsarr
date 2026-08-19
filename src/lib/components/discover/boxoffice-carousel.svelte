<script lang="ts">
	import { getBoxOfficeCurrent } from '$lib/api/boxarr';
	import { fromBoxarrMovie, type MediaItem } from '$lib/api/media';
	import Carousel from '$lib/components/media/carousel.svelte';

	let { onSelect }: { onSelect?: (item: MediaItem) => void } = $props();

	let items: MediaItem[] = $state([]);
	let loading = $state(true);
	let error: Error | null = $state(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const movies = await getBoxOfficeCurrent();
			items = (movies || []).map(fromBoxarrMovie);
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	load();
</script>

<Carousel title="Box Office" {items} {loading} {error} hasMore={false} {onSelect} onRetry={load} />

<script lang="ts">
	import type { MediaItem } from '$lib/api/media';
	import BoxofficeCarousel from '$lib/components/discover/boxoffice-carousel.svelte';
	import SuggestionsCarousel from '$lib/components/discover/suggestions-carousel.svelte';
	import SeerrCarousel from '$lib/components/discover/seerr-carousel.svelte';
	import MediaDetailModal from '$lib/components/media/media-detail-modal.svelte';

	let selected: MediaItem | null = $state(null);
</script>

<svelte:head>
	<meta name="description" content="TipsArr Discover" />
</svelte:head>

<div class="grid gap-8">
	<h1 class="font-bold text-2xl">Discover</h1>

	<BoxofficeCarousel onSelect={(item) => (selected = item)} />
	<SuggestionsCarousel onSelect={(item) => (selected = item)} />
	<SeerrCarousel title="Trending" source="trending" onSelect={(item) => (selected = item)} />
	<SeerrCarousel title="Discover Movies" source="movies" onSelect={(item) => (selected = item)} />
	<SeerrCarousel title="Discover TV" source="tv" onSelect={(item) => (selected = item)} />
</div>

<MediaDetailModal open={selected !== null} item={selected} onclose={() => (selected = null)} />

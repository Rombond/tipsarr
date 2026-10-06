<script lang="ts">
	import { api, unwrap, type MediaItem } from '$lib/api/client';
	import DiscoverRow from '$lib/components/discover/discover-row.svelte';
	import SuggestionRows from '$lib/components/discover/suggestion-rows.svelte';
	import BoxofficeRow from '$lib/components/discover/boxoffice-row.svelte';
	import MediaDetailModal from '$lib/components/media/media-detail-modal.svelte';

	let selected: MediaItem | null = $state(null);
	const select = (item: MediaItem) => (selected = item);
</script>

<svelte:head>
	<title>Discover · Tipsarr</title>
</svelte:head>

<div class="grid gap-8">
	<h1 class="font-bold text-2xl">Discover</h1>

	<SuggestionRows onSelect={select} />
	<BoxofficeRow />

	<DiscoverRow
		title="Trending"
		onSelect={select}
		load={(page) => unwrap(api.GET('/discover/trending', { params: { query: { page } } }))}
	/>
	<DiscoverRow
		title="Popular movies"
		onSelect={select}
		load={(page) => unwrap(api.GET('/discover/movies', { params: { query: { page } } }))}
	/>
	<DiscoverRow
		title="Popular TV"
		onSelect={select}
		load={(page) => unwrap(api.GET('/discover/tv', { params: { query: { page } } }))}
	/>
	<DiscoverRow
		title="Upcoming movies"
		onSelect={select}
		load={(page) => unwrap(api.GET('/discover/upcoming', { params: { query: { page } } }))}
	/>
</div>

<MediaDetailModal open={selected !== null} item={selected} onclose={() => (selected = null)} />

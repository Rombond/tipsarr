<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, type MediaItem } from '$lib/api/client';
	import DiscoverRow from '$lib/components/discover/discover-row.svelte';
	import SuggestionRows from '$lib/components/discover/suggestion-rows.svelte';
	import BoxofficeRow from '$lib/components/discover/boxoffice-row.svelte';
	import Hero from '$lib/components/discover/hero.svelte';
	import SetupChecklist from '$lib/components/discover/setup-checklist.svelte';
	import MediaDetailModal from '$lib/components/media/media-detail-modal.svelte';

	let selected: MediaItem | null = $state(null);
	const select = (item: MediaItem) => (selected = item);
</script>

{#snippet boxoffice()}<BoxofficeRow />{/snippet}
{#snippet trending()}
	<DiscoverRow href="/discover/trending" description={t('discover.trending_desc')} title={t('discover.trending')} onSelect={select} load={(page) => unwrap(api.GET('/discover/trending', { params: { query: { page } } }))} />
{/snippet}
{#snippet popularMovies()}
	<DiscoverRow href="/discover/popular-movies" description={t('discover.popular_desc')} title={t('discover.popular_movies')} onSelect={select} load={(page) => unwrap(api.GET('/discover/movies', { params: { query: { page } } }))} />
{/snippet}
{#snippet popularTv()}
	<DiscoverRow href="/discover/popular-tv" description={t('discover.popular_desc')} title={t('discover.popular_tv')} onSelect={select} load={(page) => unwrap(api.GET('/discover/tv', { params: { query: { page } } }))} />
{/snippet}
{#snippet upcoming()}
	<DiscoverRow href="/discover/upcoming" description={t('discover.upcoming_desc')} title={t('discover.upcoming')} onSelect={select} load={(page) => unwrap(api.GET('/discover/upcoming', { params: { query: { page } } }))} />
{/snippet}

<svelte:head>
	<title>{t('discover.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-8">
	<h1 class="sr-only">{t('discover.title')}</h1>
	<SetupChecklist />
	<Hero />

	<SuggestionRows onSelect={select} fillers={[boxoffice, trending, popularMovies, popularTv, upcoming]} />
</div>

<MediaDetailModal open={selected !== null} item={selected} onclose={() => (selected = null)} />

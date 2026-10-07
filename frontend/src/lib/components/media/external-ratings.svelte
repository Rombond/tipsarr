<script lang="ts">
	// IMDb, Metacritic and Rotten Tomatoes scores of a movie, read through Radarr. Nothing is shown
	// while loading, when there is no Radarr, or when Radarr has no score: the page does not change.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { fmtNumber } from '$lib/i18n/format';

	let { tmdbId }: { tmdbId: number } = $props();

	let scores = $state<Schemas['MovieScores'] | null>(null);

	$effect(() => {
		const id = tmdbId;
		scores = null;
		unwrap(api.GET('/media/movie/{id}/ratings', { params: { path: { id } } }))
			.then((s) => {
				if (id === tmdbId) scores = s;
			})
			.catch(() => {});
	});

	// Metacritic's own colour bands; Rotten Tomatoes: fresh from 60 %
	const metaTone = (v: number) => (v >= 61 ? 'bg-emerald-600' : v >= 40 ? 'bg-amber-500' : 'bg-red-600');
	const rtTone = (v: number) => (v >= 60 ? 'bg-red-600' : 'bg-emerald-700');
</script>

{#if scores && (scores.imdb || scores.metacritic || scores.rottenTomatoes)}
	<div class="flex flex-wrap items-stretch gap-2 rounded-xl border border-border p-3 text-sm" role="group" aria-label={t('ratings.title')}>
		{#if scores.imdb}
			{@const imdb = scores.imdb}
			<svelte:element
				this={scores.imdbUrl ? 'a' : 'div'}
				href={scores.imdbUrl || undefined}
				target={scores.imdbUrl ? '_blank' : undefined}
				rel={scores.imdbUrl ? 'noreferrer' : undefined}
				class="flex items-center gap-2 rounded-lg px-1 {scores.imdbUrl ? 'hover:bg-accent' : ''}"
				title={imdb.votes ? t('ratings.votes', { count: fmtNumber(imdb.votes) }) : undefined}
			>
				<span class="rounded bg-[#f5c518] px-1.5 py-0.5 text-xs font-extrabold text-black">IMDb</span>
				<span class="font-semibold tabular-nums">{imdb.value.toFixed(1)}<span class="font-normal text-muted-foreground"> / 10</span></span>
			</svelte:element>
		{/if}
		{#if scores.rottenTomatoes}
			{@const rt = scores.rottenTomatoes}
			<div class="flex items-center gap-2 px-1" title={t('ratings.rt_hint')}>
				<span class="rounded px-1.5 py-0.5 text-xs font-extrabold text-white {rtTone(rt.value)}">RT</span>
				<span class="font-semibold tabular-nums">{Math.round(rt.value)}%</span>
			</div>
		{/if}
		{#if scores.metacritic}
			{@const mc = scores.metacritic}
			<div class="flex items-center gap-2 px-1" title={t('ratings.mc_hint')}>
				<span class="min-w-7 rounded px-1.5 py-0.5 text-center text-xs font-extrabold text-white tabular-nums {metaTone(mc.value)}">{Math.round(mc.value)}</span>
				<span class="font-semibold">Metacritic</span>
			</div>
		{/if}
	</div>
{/if}

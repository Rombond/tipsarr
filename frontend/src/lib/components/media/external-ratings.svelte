<script lang="ts">
	// All the scores of a title on one line, each a link to its source (like Seerr): TMDB for every
	// title, and for movies IMDb, Rotten Tomatoes and Metacritic as read through Radarr. Scores that
	// are missing are left out; the box does not show while there is nothing to show.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { fmtNumber } from '$lib/i18n/format';
	import PopcornIcon from '@lucide/svelte/icons/popcorn';

	let { tmdbId, type, title, tmdbScore, tmdbVotes }: { tmdbId: number; type: 'movie' | 'tv'; title: string; tmdbScore?: number; tmdbVotes?: number } = $props();

	let scores = $state<Schemas['MovieScores'] | null>(null);

	$effect(() => {
		const id = tmdbId;
		scores = null;
		// movies: IMDb, Metacritic and Rotten Tomatoes (Radarr first, Rotten Tomatoes' own search as a
		// fallback); shows: Rotten Tomatoes only
		const req = type === 'movie' ? api.GET('/media/movie/{id}/ratings', { params: { path: { id } } }) : api.GET('/media/tv/{id}/ratings', { params: { path: { id } } });
		unwrap(req)
			.then((s) => {
				if (id === tmdbId) scores = s;
			})
			.catch(() => {});
	});

	// Metacritic's own colour bands; Rotten Tomatoes: fresh from 60 %
	const metaTone = (v: number) => (v >= 61 ? 'bg-emerald-600' : v >= 40 ? 'bg-amber-500' : 'bg-red-600');
	const rtTone = (v: number) => (v >= 60 ? 'bg-red-600' : 'bg-emerald-700');
	const q = $derived(encodeURIComponent(title));
	// Metacritic has no id in our data (its search page); Rotten Tomatoes' own page when its search found it
	const rtUrl = $derived(scores?.rottenTomatoesUrl || `https://www.rottentomatoes.com/search?search=${q}`);
	const mcUrl = $derived(`https://www.metacritic.com/search/${q}/`);
	const tmdbUrl = $derived(`https://www.themoviedb.org/${type}/${tmdbId}`);

	const link = 'inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring';
</script>

{#if tmdbScore || scores?.imdb || scores?.metacritic || scores?.rottenTomatoes || scores?.rottenTomatoesAudience}
	<div class="flex flex-wrap items-center gap-x-1 gap-y-0.5 rounded-xl border border-border px-1.5 py-1 text-sm" role="group" aria-label={t('ratings.title')}>
		{#if tmdbScore}
			<a class={link} href={tmdbUrl} target="_blank" rel="noreferrer" title={tmdbVotes ? t('ratings.votes', { count: fmtNumber(tmdbVotes) }) : 'TMDB'}>
				<span class="rounded bg-[#01b4e4] px-1.5 py-0.5 text-[11px] leading-4 font-extrabold text-white">TMDB</span>
				<span class="font-semibold tabular-nums">{tmdbScore.toFixed(1)}</span>
			</a>
		{/if}
		{#if scores?.imdb}
			<a class={link} href={scores.imdbUrl || `https://www.imdb.com/find/?q=${q}`} target="_blank" rel="noreferrer" title={scores.imdb.votes ? t('ratings.votes', { count: fmtNumber(scores.imdb.votes) }) : 'IMDb'}>
				<span class="rounded bg-[#f5c518] px-1.5 py-0.5 text-[11px] leading-4 font-extrabold text-black">IMDb</span>
				<span class="font-semibold tabular-nums">{scores.imdb.value.toFixed(1)}</span>
			</a>
		{/if}
		{#if scores?.rottenTomatoes}
			<a class={link} href={rtUrl} target="_blank" rel="noreferrer" title={t('ratings.rt_hint')}>
				<span class="rounded px-1.5 py-0.5 text-[11px] leading-4 font-extrabold text-white {rtTone(scores.rottenTomatoes.value)}">RT</span>
				<span class="font-semibold tabular-nums">{Math.round(scores.rottenTomatoes.value)}%</span>
			</a>
		{/if}
		{#if scores?.rottenTomatoesAudience}
			<a class={link} href={rtUrl} target="_blank" rel="noreferrer" title={t('ratings.rt_audience_hint')}>
				<span class="inline-flex items-center gap-0.5 rounded px-1.5 py-0.5 text-[11px] leading-4 font-extrabold text-white {rtTone(scores.rottenTomatoesAudience.value)}"><PopcornIcon class="size-3" />RT</span>
				<span class="font-semibold tabular-nums">{Math.round(scores.rottenTomatoesAudience.value)}%</span>
			</a>
		{/if}
		{#if scores?.metacritic}
			<a class={link} href={mcUrl} target="_blank" rel="noreferrer" title={t('ratings.mc_hint')}>
				<span class="rounded px-1.5 py-0.5 text-[11px] leading-4 font-extrabold text-white {metaTone(scores.metacritic.value)}">MC</span>
				<span class="font-semibold tabular-nums">{Math.round(scores.metacritic.value)}</span>
			</a>
		{/if}
	</div>
{/if}

<script lang="ts">
	// All the scores of a title on one line, each a link to its source (like Seerr, whose logos are
	// used): TMDB for every title, and for movies IMDb, Rotten Tomatoes (critics and audience) and
	// Metacritic. Scores that are missing are left out; nothing shows while there is nothing to show.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { fmtNumber } from '$lib/i18n/format';
	import { icons } from '$lib/ratings.svelte';

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

	// Metacritic shows its score in a coloured square, in its own bands
	const metaTone = (v: number) => (v >= 61 ? 'bg-emerald-600' : v >= 40 ? 'bg-amber-500' : 'bg-red-600');
	const q = $derived(encodeURIComponent(title));
	// Rotten Tomatoes' own page when its search found it, else its search page; Metacritic has no id in our data
	const rtUrl = $derived(scores?.rottenTomatoesUrl || `https://www.rottentomatoes.com/search?search=${q}`);
	const mcUrl = $derived(`https://www.metacritic.com/search/${q}/`);
	const tmdbUrl = $derived(`https://www.themoviedb.org/${type}/${tmdbId}`);

	const link = 'inline-flex shrink-0 items-center gap-1.5 rounded-md px-1.5 py-1 whitespace-nowrap hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring';
	const logo = 'h-5 w-auto max-w-9';
</script>

{#if tmdbScore || scores?.imdb || scores?.metacritic || scores?.rottenTomatoes || scores?.rottenTomatoesAudience}
	<div class="flex flex-nowrap items-center justify-between gap-0.5 overflow-x-auto rounded-xl border border-border px-1 py-0.5 text-[13px]" role="group" aria-label={t('ratings.title')}>
		{#if scores?.rottenTomatoes}
			<a class={link} href={rtUrl} target="_blank" rel="noreferrer" title={t('ratings.rt_hint')}>
				<img src={icons.rt(scores.rottenTomatoes.value)} alt="Rotten Tomatoes" class={logo} />
				<span class="font-semibold tabular-nums">{Math.round(scores.rottenTomatoes.value)}%</span>
			</a>
		{/if}
		{#if scores?.rottenTomatoesAudience}
			<a class={link} href={rtUrl} target="_blank" rel="noreferrer" title={t('ratings.rt_audience_hint')}>
				<img src={icons.rtAudience(scores.rottenTomatoesAudience.value)} alt={t('ratings.rt_audience_hint')} class={logo} />
				<span class="font-semibold tabular-nums">{Math.round(scores.rottenTomatoesAudience.value)}%</span>
			</a>
		{/if}
		{#if scores?.imdb}
			<a class={link} href={scores.imdbUrl || `https://www.imdb.com/find/?q=${q}`} target="_blank" rel="noreferrer" title={scores.imdb.votes ? t('ratings.votes', { count: fmtNumber(scores.imdb.votes) }) : 'IMDb'}>
				<img src={icons.imdb} alt="IMDb" class={logo} />
				<span class="font-semibold tabular-nums">{scores.imdb.value.toFixed(1)}</span>
			</a>
		{/if}
		{#if scores?.metacritic}
			<a class={link} href={mcUrl} target="_blank" rel="noreferrer" title={t('ratings.mc_hint')}>
				<span class="inline-flex min-w-6 items-center justify-center rounded px-1 text-[12px] leading-5 font-extrabold text-white tabular-nums {metaTone(scores.metacritic.value)}" aria-label="Metacritic">{Math.round(scores.metacritic.value)}</span>
			</a>
		{/if}
		{#if tmdbScore}
			<a class={link} href={tmdbUrl} target="_blank" rel="noreferrer" title={tmdbVotes ? t('ratings.votes', { count: fmtNumber(tmdbVotes) }) : 'TMDB'}>
				<img src={icons.tmdb} alt="TMDB" class={logo} />
				<span class="font-semibold tabular-nums">{tmdbScore.toFixed(1)}</span>
			</a>
		{/if}
	</div>
{/if}

<script lang="ts" module>
	// In-memory session cache — survives tab switches / remounts, cleared on reload/close.
	const posterCache = new Map<string, string | null>();
</script>

<script lang="ts">
	import { getRadarrMovie, getRadarrPosterUrl, searchTmdbMoviePoster } from '$lib/api';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import ImageOffIcon from '@lucide/svelte/icons/image-off';

	let {
		radarrId,
		title,
		year,
		class: className = '',
	}: { radarrId?: number; title: string; year?: number; class?: string } = $props();

	let posterUrl: string | null = $state(null);
	let loading = $state(false);
	let failed = $state(false);

	async function loadPoster(id: number | undefined, movieTitle: string, movieYear: number | undefined) {
		const cacheKey = id ? `radarr:${id}` : `tmdb:${movieTitle}:${movieYear || ''}`;

		if (posterCache.has(cacheKey)) {
			const cached = posterCache.get(cacheKey) ?? null;
			posterUrl = cached;
			failed = !cached;
			loading = false;
			return;
		}

		posterUrl = null;
		failed = false;
		loading = true;

		try {
			if (id) {
				try {
					const movie = await getRadarrMovie(id);
					const url = getRadarrPosterUrl(movie);
					if (url) {
						posterUrl = url;
						posterCache.set(cacheKey, url);
						return;
					}
				} catch {
					// fall through to TMDB
				}
			}

			const tmdbUrl = await searchTmdbMoviePoster(movieTitle, movieYear);
			posterUrl = tmdbUrl;
			failed = !tmdbUrl;
			posterCache.set(cacheKey, tmdbUrl);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		loadPoster(radarrId, title, year);
	});
</script>

<div class="aspect-[2/3] w-full overflow-hidden bg-muted {className}">
	{#if loading}
		<Skeleton class="h-full w-full rounded-none" />
	{:else if posterUrl}
		<img src={posterUrl} alt={title} class="h-full w-full object-cover" loading="lazy" />
	{:else}
		<div class="flex h-full w-full items-center justify-center text-muted-foreground">
			<ImageOffIcon class="size-8" />
		</div>
	{/if}
</div>

<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		getMovieDetails,
		getTvDetails,
		getMovieRecommendations,
		getTvRecommendations,
		createRequest,
		posterUrl,
	} from '$lib/api/seerr';
	import { fromSeerrResult, type MediaItem } from '$lib/api/media';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Carousel from '$lib/components/media/carousel.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';

	const AVAILABILITY_LABELS: Record<number, string> = {
		1: 'Unknown',
		2: 'Pending',
		3: 'Processing',
		4: 'Partially Available',
		5: 'Available',
		6: 'Deleted',
	};

	let mediaType: 'movie' | 'tv' = $derived(page.params.type === 'tv' ? 'tv' : 'movie');
	let tmdbId = $derived(Number(page.params.id));

	let details: any = $state(null);
	let loading = $state(true);
	let error: Error | null = $state(null);

	let recommendations: MediaItem[] = $state([]);
	let recsLoading = $state(true);

	let requesting = $state(false);
	let requestError: string | null = $state(null);
	let requested = $state(false);

	async function load() {
		loading = true;
		error = null;
		requested = false;
		requestError = null;
		try {
			details = mediaType === 'tv' ? await getTvDetails(tmdbId) : await getMovieDetails(tmdbId);
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}

		recsLoading = true;
		try {
			const data =
				mediaType === 'tv' ? await getTvRecommendations(tmdbId) : await getMovieRecommendations(tmdbId);
			recommendations = (data.results || []).map((r: any) =>
				fromSeerrResult({ ...r, mediaType: r.mediaType || mediaType }),
			);
		} catch {
			recommendations = [];
		} finally {
			recsLoading = false;
		}
	}

	$effect(() => {
		if (mediaType && tmdbId) load();
	});

	function goToRecommendation(item: MediaItem) {
		goto(`/media/${item.mediaType}/${item.tmdbId}`);
	}

	async function handleRequest() {
		requesting = true;
		requestError = null;
		try {
			await createRequest({ mediaType, mediaId: tmdbId, ...(mediaType === 'tv' ? { seasons: 'all' } : {}) });
			requested = true;
		} catch (e) {
			requestError = (e as Error).message;
		} finally {
			requesting = false;
		}
	}

	let alreadyAvailable = $derived(
		details?.mediaInfo?.status === 5 || details?.mediaInfo?.status === 4,
	);
	let runtimeLabel = $derived(
		details?.runtime
			? `${Math.floor(details.runtime / 60)}h ${details.runtime % 60}m`
			: details?.episodeRunTime?.[0]
				? `${details.episodeRunTime[0]}m/ep`
				: null,
	);
	let year = $derived((details?.releaseDate || details?.firstAirDate || '').slice(0, 4));
	let cast = $derived((details?.credits?.cast || []).slice(0, 12));
	let studios = $derived(
		mediaType === 'tv' ? details?.networks : details?.productionCompanies,
	);
</script>

<svelte:head>
	<meta name="description" content="TipsArr media details" />
</svelte:head>

{#if loading}
	<div class="grid gap-4">
		<Skeleton class="h-64 w-full rounded-lg" />
		<Skeleton class="h-6 w-1/3" />
		<Skeleton class="h-4 w-2/3" />
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		Error: {error.message}
		<button class="ml-2 underline" onclick={load}>Retry</button>
	</div>
{:else if details}
	<div class="-mx-4 -mt-4 md:-mx-6 md:-mt-6">
		<div class="relative">
			<div class="h-56 w-full overflow-hidden md:h-80">
				{#if details.backdropPath}
					<img src={posterUrl(details.backdropPath)} alt="" class="h-full w-full object-cover" />
					<div class="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent"></div>
				{:else}
					<div class="h-full w-full bg-muted"></div>
				{/if}
			</div>
			<button
				type="button"
				class="absolute top-4 left-4 z-10 flex items-center gap-1.5 rounded-full bg-background/80 px-3 py-1.5 text-sm shadow hover:bg-background"
				onclick={() => history.back()}
			>
				<ArrowLeftIcon class="size-4" />
				Back
			</button>

			<div class="relative z-10 -mt-16 flex flex-col gap-4 px-4 sm:flex-row sm:items-end md:-mt-24 md:px-6">
				<img
					src={posterUrl(details.posterPath)}
					alt={details.title || details.name}
					class="w-32 shrink-0 rounded-lg shadow-lg sm:w-44"
				/>
				<div class="grid gap-2 pb-1">
					<h1 class="font-bold text-2xl text-foreground drop-shadow md:text-3xl">{details.title || details.name}</h1>
					<div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
						<Badge variant="secondary">{mediaType === 'tv' ? 'TV' : 'Movie'}</Badge>
						{#if year}<span>{year}</span>{/if}
						{#if runtimeLabel}<span>· {runtimeLabel}</span>{/if}
						{#if details.voteAverage}<span>· ★ {details.voteAverage.toFixed(1)}</span>{/if}
					</div>
				</div>
			</div>
		</div>

		<div class="grid gap-6 px-4 pb-6 md:grid-cols-3 md:px-6">
			<div class="grid gap-6 md:col-span-2">
				{#if details.genres?.length}
					<div class="flex flex-wrap gap-1.5">
						{#each details.genres as genre (genre.id)}
							<Badge variant="outline">{genre.name}</Badge>
						{/each}
					</div>
				{/if}

				{#if details.mediaInfo?.status}
					<Badge variant="outline" class="w-fit">{AVAILABILITY_LABELS[details.mediaInfo.status] || 'Unknown'}</Badge>
				{/if}

				{#if details.tagline}
					<p class="text-sm text-muted-foreground italic">{details.tagline}</p>
				{/if}

				{#if details.overview}
					<p class="max-w-3xl text-sm leading-relaxed">{details.overview}</p>
				{/if}

				{#if requestError}
					<p class="text-sm text-destructive">{requestError}</p>
				{/if}

				<div class="flex flex-wrap gap-2">
					<Button onclick={handleRequest} disabled={requesting || requested || alreadyAvailable}>
						{#if alreadyAvailable}
							Already available
						{:else if requested}
							Requested
						{:else if requesting}
							Requesting…
						{:else}
							Request
						{/if}
					</Button>
					{#if details.externalIds?.imdbId}
						<Button
							variant="outline"
							href="https://www.imdb.com/title/{details.externalIds.imdbId}"
							target="_blank"
							rel="noreferrer"
						>
							IMDb
							<ExternalLinkIcon data-icon="inline-end" />
						</Button>
					{/if}
					<Button
						variant="outline"
						href="https://www.themoviedb.org/{mediaType}/{tmdbId}"
						target="_blank"
						rel="noreferrer"
					>
						TMDB
						<ExternalLinkIcon data-icon="inline-end" />
					</Button>
				</div>

				{#if cast.length}
					<section class="grid gap-2">
						<h2 class="font-semibold text-lg">Cast</h2>
						<div class="no-scrollbar flex gap-3 overflow-x-auto pb-2">
							{#each cast as member (member.id)}
								<button
									type="button"
									class="w-24 shrink-0 text-center"
									onclick={() => goto(`/person/${member.id}`)}
								>
									<div class="aspect-square w-full overflow-hidden rounded-full bg-muted">
										{#if member.profilePath}
											<img src={posterUrl(member.profilePath)} alt={member.name} class="h-full w-full object-cover" loading="lazy" />
										{/if}
									</div>
									<p class="mt-1 truncate text-xs font-medium">{member.name}</p>
									<p class="truncate text-[10px] text-muted-foreground">{member.character}</p>
								</button>
							{/each}
						</div>
					</section>
				{/if}

				<Carousel
					title="More like this"
					items={recommendations}
					loading={recsLoading}
					hasMore={false}
					onSelect={goToRecommendation}
				/>
			</div>

			<div class="grid content-start gap-4">
				{#if details.collection}
					<button
						type="button"
						class="group relative block h-28 w-full overflow-hidden rounded-lg text-left shadow-sm"
						onclick={() => goto(`/collection/${details.collection.id}`)}
					>
						{#if details.collection.backdropPath}
							<img
								src={posterUrl(details.collection.backdropPath)}
								alt={details.collection.name}
								class="h-full w-full object-cover transition-transform group-hover:scale-105"
							/>
						{:else}
							<div class="h-full w-full bg-muted"></div>
						{/if}
						<div class="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-black/80 to-transparent p-3">
							<span class="text-[10px] tracking-wide text-white/70 uppercase">Part of the collection</span>
							<span class="truncate font-semibold text-sm text-white">{details.collection.name}</span>
						</div>
					</button>
				{/if}

				<div class="grid gap-3 rounded-lg border border-border p-4 text-sm">
					<h2 class="font-semibold">Info</h2>
					{#if details.status}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Status</span>
							<span>{details.status}</span>
						</div>
					{/if}
					{#if details.originalLanguage}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Original Language</span>
							<span>{new Intl.DisplayNames(['en'], { type: 'language' }).of(details.originalLanguage) || details.originalLanguage}</span>
						</div>
					{/if}
					{#if (details.originalTitle || details.originalName) && (details.originalTitle || details.originalName) !== (details.title || details.name)}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Original Title</span>
							<span>{details.originalTitle || details.originalName}</span>
						</div>
					{/if}
					{#if studios?.length}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">{mediaType === 'tv' ? 'Network' : 'Studio'}</span>
							<span>{studios.map((s: any) => s.name).join(', ')}</span>
						</div>
					{/if}
					{#if mediaType === 'movie' && details.budget}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Budget</span>
							<span>${details.budget.toLocaleString()}</span>
						</div>
					{/if}
					{#if mediaType === 'movie' && details.revenue}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Revenue</span>
							<span>${details.revenue.toLocaleString()}</span>
						</div>
					{/if}
					{#if mediaType === 'tv' && details.numberOfSeason}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Seasons</span>
							<span>{details.numberOfSeason}</span>
						</div>
					{/if}
					{#if mediaType === 'tv' && details.numberOfEpisodes}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Episodes</span>
							<span>{details.numberOfEpisodes}</span>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.no-scrollbar {
		scrollbar-width: none;
		-ms-overflow-style: none;
	}
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
</style>

<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, type MediaDetail } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Carousel from '$lib/components/media/carousel.svelte';
	import RequestButton from '$lib/components/requests/request-button.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';

	let mediaType: 'movie' | 'tv' = $derived(page.params.type === 'tv' ? 'tv' : 'movie');
	let tmdbId = $derived(Number(page.params.id));

	let details = $state<MediaDetail | null>(null);
	let loading = $state(true);
	let error = $state<Error | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			details = await unwrap(
				api.GET('/media/{type}/{id}', { params: { path: { type: mediaType, id: tmdbId } } }),
			);
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (mediaType && tmdbId) load();
	});

	const runtimeLabel = $derived(
		details?.runtimeMinutes
			? mediaType === 'tv'
				? `${details.runtimeMinutes}m/ep`
				: `${Math.floor(details.runtimeMinutes / 60)}h ${details.runtimeMinutes % 60}m`
			: null,
	);
	const year = $derived((details?.releaseDate || '').slice(0, 4));
	const language = $derived(
		details?.originalLanguage
			? (new Intl.DisplayNames(['en'], { type: 'language' }).of(details.originalLanguage) ?? details.originalLanguage)
			: null,
	);
	const seasons = $derived((details?.seasons ?? []).filter((s) => s.number > 0));
</script>

<svelte:head>
	<title>{details?.title ?? 'Details'} · Tipsarr</title>
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
					<img src={imageUrl(details.backdropPath, 'w1280')} alt="" class="h-full w-full object-cover" />
					<div class="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent"></div>
				{:else}
					<div class="h-full w-full bg-muted"></div>
				{/if}
			</div>
			<button
				type="button"
				class="absolute top-4 left-4 z-10 flex cursor-pointer items-center gap-1.5 rounded-full bg-background/80 px-3 py-1.5 text-sm shadow hover:bg-background"
				onclick={() => history.back()}
			>
				<ArrowLeftIcon class="size-4" />
				Back
			</button>

			<div class="relative z-10 -mt-16 flex flex-col gap-4 px-4 sm:flex-row sm:items-end md:-mt-24 md:px-6">
				{#if details.posterPath}
					<img
						src={imageUrl(details.posterPath, 'w342')}
						alt={details.title}
						class="w-32 shrink-0 rounded-lg shadow-lg sm:w-44"
					/>
				{/if}
				<div class="grid gap-3 pb-1 sm:pl-2">
					<h1 class="font-bold text-2xl text-foreground drop-shadow md:text-3xl">{details.title}</h1>
					<div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
						<Badge variant="secondary">{mediaType === 'tv' ? 'TV' : 'Movie'}</Badge>
						{#if details.availability !== 'none'}
							<Badge>{details.availability === 'available' ? 'Available' : 'Partially available'}</Badge>
						{/if}
						{#if year}<span>{year}</span>{/if}
						{#if runtimeLabel}<span>· {runtimeLabel}</span>{/if}
						{#if details.voteAverage}<span>· ★ {details.voteAverage.toFixed(1)}</span>{/if}
					</div>
				</div>
			</div>
		</div>

		<div class="grid gap-x-6 gap-y-6 px-4 pt-6 pb-6 md:grid-cols-3 md:px-6">
			<!-- 1. overview + actions -->
			<div class="grid content-start gap-6 md:col-span-2">
				{#if details.genres.length}
					<div class="flex flex-wrap gap-1.5">
						{#each details.genres as genre (genre.id)}
							<Badge variant="outline">{genre.name}</Badge>
						{/each}
					</div>
				{/if}

				{#if details.tagline}
					<p class="text-sm text-muted-foreground italic">{details.tagline}</p>
				{/if}

				{#if details.overview}
					<p class="max-w-3xl text-sm leading-relaxed">{details.overview}</p>
				{/if}

				<div class="flex flex-wrap gap-2">
					<RequestButton
						type={mediaType}
						{tmdbId}
						seasons={details.seasons}
						availability={details.availability}
						requestStatus={details.requestStatus}
					/>
					{#if details.imdbId}
						<Button variant="outline" href="https://www.imdb.com/title/{details.imdbId}" target="_blank" rel="noreferrer">
							IMDb
							<ExternalLinkIcon data-icon="inline-end" />
						</Button>
					{/if}
					<Button variant="outline" href="https://www.themoviedb.org/{mediaType}/{tmdbId}" target="_blank" rel="noreferrer">
						TMDB
						<ExternalLinkIcon data-icon="inline-end" />
					</Button>
				</div>
			</div>

			<!-- 2. info sidebar: right column on desktop, before Cast on mobile -->
			<div class="grid content-start gap-4 md:col-start-3 md:row-span-2 md:row-start-1">
				{#if details.collectionId}
					<button
						type="button"
						class="group relative block h-28 w-full cursor-pointer overflow-hidden rounded-lg bg-muted text-left shadow-sm"
						onclick={() => goto(`/collection/${details!.collectionId}`)}
					>
						<div class="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-black/80 to-transparent p-3">
							<span class="text-[10px] tracking-wide text-white/70 uppercase">Part of the collection</span>
							<span class="truncate font-semibold text-sm text-white">{details.collectionName}</span>
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
					{#if language}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Original language</span>
							<span>{language}</span>
						</div>
					{/if}
					{#if details.directors.length}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Director</span>
							<span class="flex flex-wrap gap-x-2">
								{#each details.directors as d (d.id)}
									<a class="underline-offset-2 hover:underline" href="/person/{d.id}">{d.name}</a>
								{/each}
							</span>
						</div>
					{/if}
					{#if mediaType === 'tv' && details.numberOfSeasons}
						<div class="grid gap-0.5">
							<span class="text-xs text-muted-foreground">Seasons</span>
							<span>{details.numberOfSeasons}</span>
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

			<!-- 3. seasons + cast -->
			<div class="grid content-start gap-6 md:col-span-2">
				{#if mediaType === 'tv' && seasons.length}
					<section class="grid gap-2">
						<h2 class="font-semibold text-lg">Seasons</h2>
						<ul class="grid gap-1.5 text-sm">
							{#each seasons as s (s.number)}
								<li class="flex items-center justify-between rounded-md border border-border px-3 py-2">
									<span class="font-medium">{s.name || `Season ${s.number}`}</span>
									<span class="text-muted-foreground">
										{s.episodeCount} episodes{#if s.airDate} · {s.airDate.slice(0, 4)}{/if}
									</span>
								</li>
							{/each}
						</ul>
					</section>
				{/if}

				{#if details.cast.length}
					<section class="grid gap-2">
						<h2 class="font-semibold text-lg">Cast</h2>
						<div class="no-scrollbar flex gap-3 overflow-x-auto pb-2">
							{#each details.cast as member (member.id)}
								<button
									type="button"
									class="w-24 shrink-0 cursor-pointer text-center"
									onclick={() => goto(`/person/${member.id}`)}
								>
									<div class="aspect-square w-full overflow-hidden rounded-full bg-muted">
										{#if member.profilePath}
											<img src={imageUrl(member.profilePath, 'w185')} alt={member.name} class="h-full w-full object-cover" loading="lazy" />
										{/if}
									</div>
									<p class="mt-1 truncate text-xs font-medium">{member.name}</p>
									<p class="truncate text-[10px] text-muted-foreground">{member.character}</p>
								</button>
							{/each}
						</div>
					</section>
				{/if}
			</div>

			<!-- 4. full-width recommendations -->
			<div class="md:col-span-3">
				{#if details.recommendations.length}
					<Carousel title="More like this" items={details.recommendations} />
				{/if}
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

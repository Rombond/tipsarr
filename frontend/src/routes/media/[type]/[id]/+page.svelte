<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { hasKey } from '$lib/i18n/index.svelte';
	import { fmtLongDate, fmtMoney, languageName } from '$lib/i18n/format';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, errorText, type MediaDetail } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Carousel from '$lib/components/media/carousel.svelte';
	import Scroller from '$lib/components/ui/scroller.svelte';
	import DetailActions from '$lib/components/media/detail-actions.svelte';
	import SeasonList from '$lib/components/media/season-list.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import StarIcon from '@lucide/svelte/icons/star';

	let mediaType: 'movie' | 'tv' = $derived(page.params.type === 'tv' ? 'tv' : 'movie');
	let tmdbId = $derived(Number(page.params.id));

	let details = $state<MediaDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			details = await unwrap(api.GET('/media/{type}/{id}', { params: { path: { type: mediaType, id: tmdbId } } }));
		} catch (e) {
			error = errorText(e);
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
				? t('detail.per_episode', { minutes: details.runtimeMinutes })
				: t('time.hours', { h: Math.floor(details.runtimeMinutes / 60), m: details.runtimeMinutes % 60 })
			: null,
	);
	const year = $derived((details?.releaseDate || '').slice(0, 4));
	const language = $derived(
		details?.originalLanguage
			? languageName(details.originalLanguage)
			: null,
	);
	const seasons = $derived((details?.seasons ?? []).filter((s) => s.number > 0));
	const money = (n?: number) => (n ? fmtMoney(n) : null);
	const fullDate = (d?: string) => fmtLongDate(d) || null;
	const statusLabel = (s?: string) => (s && hasKey(`status.${s}`) ? t(`status.${s}` as 'status.Released') : (s ?? null));

	// label/value rows of the info panel; empty values are skipped
	const facts = $derived(
		details
			? ([
					[t('fact.release_date'), fullDate(details.releaseDate)],
					[t('fact.runtime'), runtimeLabel],
					[t('fact.status'), statusLabel(details.status)],
					[t('fact.language'), language],
					mediaType === 'tv' ? [t('fact.seasons'), details.numberOfSeasons ? `${details.numberOfSeasons}` : null] : [t('fact.budget'), money(details.budget)],
					mediaType === 'tv' ? [t('fact.episodes'), details.numberOfEpisodes ? `${details.numberOfEpisodes}` : null] : [t('fact.revenue'), money(details.revenue)],
					[mediaType === 'tv' ? t('fact.network') : t('fact.studio'), details.studios?.slice(0, 3).join(', ') || null],
				] as [string, string | null][]).filter(([, v]) => v)
			: [],
	);
</script>

<svelte:head>
	<title>{details?.title ?? t('detail.title_fallback')} · Tipsarr</title>
</svelte:head>

{#if loading}
	<div class="-mx-4 -mt-20 grid gap-4 md:-mx-8">
		<Skeleton class="h-72 w-full rounded-none md:h-96" />
		<div class="grid gap-3 px-4 md:px-8">
			<Skeleton class="h-8 w-1/3" />
			<Skeleton class="h-4 w-2/3" />
		</div>
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		{t('common.error_prefix', { message: error })}
		<button class="ml-2 underline" onclick={load}>{t('common.retry')}</button>
	</div>
{:else if details}
	<!-- the backdrop runs under the floating search bar (-mt-20 cancels the page's top padding) -->
	<div class="-mx-4 -mt-20 md:-mx-8">
		<div class="relative">
			<div class="h-72 w-full overflow-hidden md:h-[26rem]">
				{#if details.backdropPath}
					<img src={imageUrl(details.backdropPath, 'w1280')} alt="" class="h-full w-full object-cover" />
				{:else}
					<div class="h-full w-full bg-muted"></div>
				{/if}
				<div class="absolute inset-0 bg-gradient-to-t from-background via-background/70 to-background/10"></div>
			</div>
			<button
				type="button"
				class="absolute top-20 left-4 z-10 flex cursor-pointer items-center gap-1.5 rounded-full bg-background/70 px-3 py-1.5 text-sm shadow backdrop-blur hover:bg-background md:left-8"
				onclick={() => history.back()}
			>
				<ArrowLeftIcon class="size-4" />
				{t('common.back')}
			</button>

			<div class="absolute inset-x-0 bottom-0 flex items-end gap-4 px-4 md:gap-6 md:px-8">
				{#if details.posterPath}
					<img
						src={imageUrl(details.posterPath, 'w342')}
						alt={details.title}
						class="hidden w-28 shrink-0 translate-y-10 rounded-lg shadow-xl ring-1 ring-border/50 sm:block sm:w-40 md:w-52"
					/>
				{/if}
				<div class="grid gap-2 pb-4">
					<h1 class="font-bold text-2xl drop-shadow sm:text-3xl md:text-5xl">
						{details.title}{#if year}<span class="ml-2 font-normal text-muted-foreground">({year})</span>{/if}
					</h1>
					<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
						<Badge variant="secondary">{mediaType === 'tv' ? t('type.tv_short') : t('type.movie')}</Badge>
						{#if details.availability !== 'none'}<Badge>{details.availability === 'available' ? t('media.available') : t('media.partially_available')}</Badge>{/if}
						{#if runtimeLabel}<span>{runtimeLabel}</span>{/if}
						{#if details.voteAverage}<span class="inline-flex items-center gap-1"><StarIcon class="size-4 fill-amber-400 text-amber-400" />{details.voteAverage.toFixed(1)}</span>{/if}
						{#if details.genres.length}<span>{details.genres.map((g) => g.name).join(' · ')}</span>{/if}
					</div>
				</div>
			</div>
		</div>

		<div class="grid gap-x-10 gap-y-8 px-4 pt-6 pb-8 sm:pt-14 md:px-8 lg:grid-cols-[minmax(0,1fr)_21rem]">
			<!-- overview and actions -->
			<div class="grid min-w-0 content-start gap-5 lg:col-start-1 lg:row-start-1">
				{#if details.tagline}<p class="text-muted-foreground italic">{details.tagline}</p>{/if}
				{#if details.overview}
					<div>
						<h2 class="mb-1 font-semibold text-lg">{t('detail.overview')}</h2>
						<p class="leading-7 text-foreground/90">{details.overview}</p>
					</div>
				{/if}
				<DetailActions {details} type={mediaType} {tmdbId} />
				{#if mediaType === 'tv' && seasons.length}
					<section class="grid gap-2">
						<h2 class="font-semibold text-lg">{t('seasons.title')}</h2>
						<SeasonList {tmdbId} {seasons} />
					</section>
				{/if}
			</div>

			<!-- info panel: narrow, like a spec sheet -->
			<aside class="grid content-start gap-4 lg:col-start-2 lg:row-start-1">
				{#if details.collectionId}
					<button
						type="button"
						class="group relative block h-24 w-full cursor-pointer overflow-hidden rounded-xl bg-muted text-left shadow-sm"
						onclick={() => goto(`/collection/${details!.collectionId}`)}
					>
						{#if details.collectionBackdropPath}
							<img src={imageUrl(details.collectionBackdropPath, 'w780')} alt="" class="h-full w-full object-cover transition-transform group-hover:scale-105" />
						{/if}
						<div class="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-black/85 to-black/10 p-3">
							<span class="text-[10px] tracking-wide text-white/70 uppercase">{t('detail.collection_part')}</span>
							<span class="truncate font-semibold text-sm text-white">{details.collectionName}</span>
						</div>
					</button>
				{/if}

				<dl class="rounded-xl border border-border text-sm">
					{#each facts as [label, value] (label)}
						<div class="flex items-start justify-between gap-4 border-b border-border px-4 py-2.5 last:border-b-0">
							<dt class="shrink-0 text-muted-foreground">{label}</dt>
							<dd class="text-right font-medium">{value}</dd>
						</div>
					{/each}
					{#if details.directors.length}
						<div class="flex items-start justify-between gap-4 px-4 py-2.5">
							<dt class="shrink-0 text-muted-foreground">{t('fact.director')}</dt>
							<dd class="text-right font-medium">
								{#each details.directors as d, i (d.id)}<a class="hover:underline" href="/person/{d.id}">{d.name}</a>{i < details.directors.length - 1 ? ', ' : ''}{/each}
							</dd>
						</div>
					{/if}
				</dl>
			</aside>
		</div>

		<div class="grid gap-8 px-4 pb-10 md:px-8">
			{#if details.cast.length}
				<Scroller title={t('detail.cast')}>
					{#each details.cast as member (member.id)}
						<a href="/person/{member.id}" class="w-28 shrink-0 snap-start text-center sm:w-32">
							<div class="aspect-square w-full overflow-hidden rounded-full bg-muted ring-1 ring-border/60">
								{#if member.profilePath}
									<img src={imageUrl(member.profilePath, 'w185')} alt="" class="h-full w-full object-cover object-[50%_18%]" loading="lazy" />
								{/if}
							</div>
							<p class="mt-2 line-clamp-2 text-sm font-medium leading-tight">{member.name}</p>
							{#if member.character}<p class="line-clamp-2 text-xs leading-tight text-muted-foreground">{member.character}</p>{/if}
						</a>
					{/each}
				</Scroller>
			{/if}

			{#if details.recommendations.length}
				<Carousel title={t('detail.more_like')} items={details.recommendations} />
			{/if}
		</div>
	</div>
{/if}

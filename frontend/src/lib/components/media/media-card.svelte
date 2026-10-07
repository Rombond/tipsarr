<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { goto } from '$app/navigation';
	import { imageUrl, type MediaItem } from '$lib/api/client';
	import StatusIcon from '$lib/components/ui/status-icon.svelte';
	import RequestFlow from '$lib/components/requests/request-flow.svelte';
	import { itemStatus } from '$lib/status';
	import InfoIcon from '@lucide/svelte/icons/info';
	import XIcon from '@lucide/svelte/icons/x';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import StarIcon from '@lucide/svelte/icons/star';
	import CheckIcon from '@lucide/svelte/icons/check';
	import { auth } from '$lib/stores/auth.svelte';
	import { ratings, pick, type RatingSource } from '$lib/ratings.svelte';

	let {
		item,
		onSelect,
		rank,
		note,
		inRadarr = false,
		radarrHasFile = false,
		onDismiss,
		fluid = false,
		posterUrl,
		hideStatus = false,
		watched = false,
	}: {
		item: MediaItem;
		onSelect?: (item: MediaItem) => void;
		/** Chart position, shown over the poster. */
		rank?: number;
		/** Small line under the title, e.g. a box-office gross. */
		note?: string;
		/** Radarr already tracks this movie. */
		inRadarr?: boolean;
		/** ...and already has the file. */
		radarrHasFile?: boolean;
		/** Shows a "not interested" button on hover (used by suggestion rows). */
		onDismiss?: (item: MediaItem) => void;
		/** Fill the grid cell instead of a fixed poster width. */
		fluid?: boolean;
		/** A ready poster address (a Jellyfin poster in the Library and Stats pages) instead of the TMDB path. */
		posterUrl?: string;
		/** Library and Stats only list what is available: the "available" icon would be on every card. */
		hideStatus?: boolean;
		/** Shows a check when the signed-in person already watched it (Library). */
		watched?: boolean;
	} = $props();

	let requested = $state<string | null>(null);
	let requesting = $state(false);
	let flow: { start: () => void } | undefined = $state();

	const poster = $derived(posterUrl ?? imageUrl(item.posterPath, 'w342'));
	const year = $derived(item.releaseDate?.slice(0, 4));
	const shown = $derived(itemStatus({ availability: item.availability, requestStatus: requested ?? item.requestStatus }, { tracked: inRadarr, hasFile: radarrHasFile }));
	const canQuickRequest = $derived(item.availability !== 'available' && shown === null);

	// the score on the poster follows the person's choice (movies only; shows keep TMDB)
	const source = $derived((auth.user?.ratingSource || 'tmdb') as RatingSource);
	$effect(() => {
		if (source !== 'tmdb' && item.type === 'movie') ratings.want(item.tmdbId, source === 'rottenTomatoes');
	});
	const external = $derived(source !== 'tmdb' && item.type === 'movie' ? pick(source, ratings.scores[item.tmdbId]) : null);

	function openDetails() {
		goto(`/media/${item.type}/${item.tmdbId}`);
	}
</script>

<RequestFlow bind:this={flow} bind:busy={requesting} type={item.type} tmdbId={item.tmdbId} title={item.title} onrequested={(r) => (requested = r.status)} />

<div class={fluid ? 'min-w-0' : 'w-32 shrink-0 snap-start sm:w-40'}>
	<div
		class="group relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm ring-1 ring-border/50 transition hover:shadow-lg hover:ring-border focus-within:ring-2 focus-within:ring-ring"
	>
		<button
			type="button"
			class="block h-full w-full cursor-pointer text-left"
			aria-label={item.title}
			onclick={() => (onSelect ? onSelect(item) : openDetails())}
		>
			{#if poster}
				<img
					src={poster}
					alt=""
					class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
					loading="lazy"
				/>
			{:else}
				<div class="flex h-full w-full items-center justify-center p-2 text-center text-muted-foreground text-xs">{item.title}</div>
			{/if}
		</button>

		{#if shown && !hideStatus}
			<StatusIcon status={shown} class="absolute top-1.5 right-1.5" />
		{/if}
		{#if watched}
			<span class="pointer-events-none absolute top-1.5 right-1.5 flex size-5 items-center justify-center rounded-full bg-emerald-500 text-white shadow" title={t('library.watched_by_me')}>
				<CheckIcon class="size-3.5" />
				<span class="sr-only">{t('library.watched_by_me')}</span>
			</span>
		{/if}

		<span class="pointer-events-none absolute top-1.5 left-1.5 rounded-full bg-black/70 px-2 py-0.5 text-[10px] font-semibold tracking-wide text-white uppercase backdrop-blur-sm">
			{item.type === 'tv' ? t('type.tv_short') : t('type.movie')}
		</span>

		{#if rank}
			<span class="pointer-events-none absolute bottom-1 left-2 font-black text-4xl text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">{rank}</span>
		{/if}

		<!-- hover / focus actions (always reachable by keyboard; on touch the card opens the details) -->
		<div class="absolute inset-x-0 bottom-0 flex items-end justify-between gap-1 bg-gradient-to-t from-black/80 via-black/40 to-transparent p-1.5 pt-8 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:hidden">
			{#if canQuickRequest}
				<button
					type="button"
					class="flex cursor-pointer items-center gap-1 rounded-full bg-white/90 px-2.5 py-1 text-xs font-medium text-black shadow hover:bg-white disabled:opacity-60"
					disabled={requesting}
					onclick={(e) => {
						e.stopPropagation();
						flow?.start();
					}}
				>
					<PlusIcon class="size-3.5" />
					{requesting ? '…' : t('card.quick_request')}
				</button>
			{:else}
				<span></span>
			{/if}
			<span class="flex gap-1">
				{#if onDismiss}
					<button
						type="button"
						aria-label={t('media.not_interested')}
						title={t('media.not_interested_hint')}
						class="flex size-7 cursor-pointer items-center justify-center rounded-full bg-white/90 text-black shadow hover:bg-white"
						onclick={(e) => {
							e.stopPropagation();
							onDismiss(item);
						}}
					>
						<XIcon class="size-4" />
					</button>
				{/if}
				<button
					type="button"
					aria-label={t('media.view_details')}
					class="flex size-7 cursor-pointer items-center justify-center rounded-full bg-white/90 text-black shadow hover:bg-white"
					onclick={(e) => {
						e.stopPropagation();
						openDetails();
					}}
				>
					<InfoIcon class="size-4" />
				</button>
			</span>
		</div>
	</div>

	<p class="mt-1.5 truncate text-sm font-medium" title={item.title}>{item.title}</p>
	<p class="flex items-center gap-1.5 truncate text-xs text-muted-foreground">
		{#if year}<span>{year}</span>{/if}
		{#if external}
			<span class="inline-flex items-center gap-1" title={t('card.rating_source_hint', { source: external.label })}><span class="rounded px-1 text-[9px] leading-4 font-extrabold {external.tone}">{external.label}</span>{external.text}</span>
		{:else if item.voteAverage}
			<span class="inline-flex items-center gap-0.5" title={t('card.rating_hint')}><StarIcon class="size-3 fill-amber-400 text-amber-400" />{item.voteAverage.toFixed(1)}</span>
		{/if}
	</p>
	{#if note}<p class="truncate text-xs text-muted-foreground">{note}</p>{/if}
</div>

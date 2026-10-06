<script lang="ts">
	import { goto } from '$app/navigation';
	import { imageUrl, type MediaItem } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import InfoIcon from '@lucide/svelte/icons/info';

	let {
		item,
		onSelect,
		rank,
		note,
		inRadarr = false,
	}: {
		item: MediaItem;
		onSelect?: (item: MediaItem) => void;
		/** Chart position, shown over the poster. */
		rank?: number;
		/** Small line under the title, e.g. a box-office gross. */
		note?: string;
		/** Radarr already tracks this movie. */
		inRadarr?: boolean;
	} = $props();

	const poster = $derived(imageUrl(item.posterPath, 'w342'));
	const year = $derived(item.releaseDate?.slice(0, 4));

	function openDetails() {
		goto(`/media/${item.type}/${item.tmdbId}`);
	}
</script>

<div class="w-40 shrink-0 snap-start">
	<div class="group relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm transition-shadow hover:shadow-md">
		<button
			type="button"
			class="block h-full w-full cursor-pointer text-left"
			onclick={() => (onSelect ? onSelect(item) : openDetails())}
		>
			{#if poster}
				<img src={poster} alt={item.title} class="h-full w-full object-cover" loading="lazy" />
			{:else}
				<div class="flex h-full w-full items-center justify-center text-muted-foreground text-xs">No image</div>
			{/if}
		</button>
		{#if item.availability !== 'none'}
			<Badge class="absolute top-1.5 left-1.5 text-[10px]">
				{item.availability === 'available' ? 'Available' : 'Partial'}
			</Badge>
		{:else if item.requestStatus}
			<Badge variant="secondary" class="absolute top-1.5 left-1.5 text-[10px]">
				{item.requestStatus === 'pending' ? 'Requested' : 'Approved'}
			</Badge>
		{:else if inRadarr}
			<Badge variant="secondary" class="absolute top-1.5 left-1.5 text-[10px]">In Radarr</Badge>
		{/if}
		{#if rank}
			<span class="pointer-events-none absolute bottom-1 left-2 font-black text-4xl text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">{rank}</span>
		{/if}
		<button
			type="button"
			aria-label="View details"
			class="absolute top-1.5 right-1.5 flex size-7 cursor-pointer items-center justify-center rounded-full bg-background/80 text-foreground opacity-0 shadow transition-opacity group-hover:opacity-100 hover:bg-background"
			onclick={openDetails}
		>
			<InfoIcon class="size-4" />
		</button>
	</div>
	<div class="mt-1.5 flex items-center gap-1.5">
		<Badge variant="secondary" class="text-[10px]">{item.type === 'tv' ? 'TV' : 'Movie'}</Badge>
		{#if item.voteAverage}
			<span class="text-xs text-muted-foreground">★ {item.voteAverage.toFixed(1)}</span>
		{/if}
		{#if year}
			<span class="text-xs text-muted-foreground">{year}</span>
		{/if}
	</div>
	<p class="mt-0.5 truncate text-sm font-medium">{item.title}</p>
	{#if note}<p class="truncate text-xs text-muted-foreground">{note}</p>{/if}
</div>

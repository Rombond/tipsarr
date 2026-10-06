<script lang="ts">
	import { goto } from '$app/navigation';
	import type { MediaItem } from '$lib/api/media';
	import { Badge } from '$lib/components/ui/badge';
	import MoviePoster from '$lib/components/boxoffice/movie-poster.svelte';
	import InfoIcon from '@lucide/svelte/icons/info';

	const AVAILABILITY_LABELS: Record<number, string> = {
		1: 'Unknown',
		2: 'Pending',
		3: 'Processing',
		4: 'Partially Available',
		5: 'Available',
		6: 'Deleted',
	};

	let { item, onSelect }: { item: MediaItem; onSelect?: (item: MediaItem) => void } = $props();

	function handleViewDetails() {
		if (item.tmdbId) {
			goto(`/media/${item.mediaType}/${item.tmdbId}`);
		} else {
			onSelect?.(item);
		}
	}
</script>

<div class="w-40 shrink-0 snap-start">
	<div class="group relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm transition-shadow hover:shadow-md">
		<button type="button" class="block h-full w-full text-left" onclick={() => onSelect?.(item)}>
			{#if item.source === 'boxarr'}
				<MoviePoster radarrId={(item.raw as any).radarr_id} title={item.title} />
			{:else if item.posterUrl}
				<img src={item.posterUrl} alt={item.title} class="h-full w-full object-cover" loading="lazy" />
			{:else}
				<div class="flex h-full w-full items-center justify-center text-muted-foreground text-xs">No image</div>
			{/if}
		</button>
		<button
			type="button"
			aria-label="View details"
			class="absolute top-1.5 right-1.5 flex size-7 items-center justify-center rounded-full bg-background/80 text-foreground opacity-0 shadow transition-opacity group-hover:opacity-100 hover:bg-background"
			onclick={handleViewDetails}
		>
			<InfoIcon class="size-4" />
		</button>
	</div>
	<div class="mt-1.5 flex items-center gap-1.5">
		<Badge variant="secondary" class="text-[10px]">{item.mediaType === 'tv' ? 'TV' : 'Movie'}</Badge>
		{#if item.rating}
			<span class="text-xs text-muted-foreground">★ {item.rating.toFixed(1)}</span>
		{/if}
	</div>
	<p class="mt-0.5 truncate text-sm font-medium">{item.title}</p>
	{#if item.availabilityStatus}
		<Badge variant="outline" class="mt-1 text-[10px]">{AVAILABILITY_LABELS[item.availabilityStatus] || 'Unknown'}</Badge>
	{/if}
</div>

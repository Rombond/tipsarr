<script lang="ts">
	import type { MediaItem } from '$lib/api/client';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import MediaCard from './media-card.svelte';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	let {
		title,
		items = [],
		loading = false,
		error = null,
		hasMore = false,
		onLoadMore,
		onSelect,
		onRetry,
		onDismiss,
	}: {
		title: string;
		items?: MediaItem[];
		loading?: boolean;
		error?: Error | null;
		hasMore?: boolean;
		onLoadMore?: () => void;
		onSelect?: (item: MediaItem) => void;
		onRetry?: () => void;
		onDismiss?: (item: MediaItem) => void;
	} = $props();

	let scrollEl: HTMLDivElement | undefined = $state();

	function handleScroll(e: Event) {
		if (!hasMore || loading) return;
		const el = e.currentTarget as HTMLDivElement;
		if (el.scrollLeft + el.clientWidth >= el.scrollWidth - 400) {
			onLoadMore?.();
		}
	}

	function scrollBy(direction: 1 | -1) {
		scrollEl?.scrollBy({ left: direction * scrollEl.clientWidth * 0.9, behavior: 'smooth' });
	}
</script>

<section class="grid gap-2">
	<div class="flex items-center justify-between">
		<h2 class="font-semibold text-lg">{title}</h2>
		<div class="flex gap-1 [@media(hover:none)]:hidden">
			<Button variant="outline" size="icon-sm" aria-label="Scroll left" onclick={() => scrollBy(-1)}>
				<ChevronLeftIcon />
			</Button>
			<Button variant="outline" size="icon-sm" aria-label="Scroll right" onclick={() => scrollBy(1)}>
				<ChevronRightIcon />
			</Button>
		</div>
	</div>
	{#if error}
		<div class="text-sm text-destructive">
			Error: {error.message}
			<button class="ml-2 underline" onclick={() => onRetry?.()}>Retry</button>
		</div>
	{:else}
		<div bind:this={scrollEl} class="no-scrollbar -mx-4 flex snap-x scroll-px-4 gap-3 overflow-x-auto px-4 pb-2 md:-mx-8 md:scroll-px-8 md:px-8" onscroll={handleScroll}>
			{#each items as item (`${item.type}:${item.tmdbId}`)}
				<MediaCard {item} {onSelect} {onDismiss} />
			{/each}
			{#if loading}
				{#each { length: 6 } as _, i (i)}
					<div class="w-40 shrink-0">
						<Skeleton class="aspect-[2/3] w-full rounded-lg" />
						<Skeleton class="mt-1.5 h-3 w-3/4" />
					</div>
				{/each}
			{/if}
			{#if !loading && items.length === 0}
				<p class="text-muted-foreground text-sm">No results found.</p>
			{/if}
		</div>
	{/if}
</section>

<style>
	.no-scrollbar {
		scrollbar-width: none;
		-ms-overflow-style: none;
	}
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
</style>

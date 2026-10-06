<script lang="ts">
	import { goto } from '$app/navigation';
	import { imageUrl, type MediaItem } from '$lib/api/client';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';

	let {
		open = false,
		item = null,
		onclose,
	}: { open?: boolean; item?: MediaItem | null; onclose?: () => void } = $props();

	function handleOpenChange(next: boolean) {
		if (!next) onclose?.();
	}

	function handleViewDetails() {
		if (!item) return;
		const url = `/media/${item.type}/${item.tmdbId}`;
		onclose?.();
		goto(url);
	}
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
	<Dialog.Content>
		{#if item}
			<Dialog.Header>
				<div class="flex items-center gap-2">
					<Badge variant="secondary">{item.type === 'tv' ? 'TV' : 'Movie'}</Badge>
					{#if item.voteAverage}
						<span class="text-xs text-muted-foreground">★ {item.voteAverage.toFixed(1)}</span>
					{/if}
					{#if item.availability !== 'none'}
						<Badge>{item.availability === 'available' ? 'Available' : 'Partial'}</Badge>
					{/if}
					{#if item.releaseDate}
						<span class="text-xs text-muted-foreground">{item.releaseDate}</span>
					{/if}
				</div>
				<Dialog.Title>{item.title}</Dialog.Title>
			</Dialog.Header>
			<div class="grid gap-2">
				{#if item.posterPath}
					<img
						src={imageUrl(item.posterPath, 'w342')}
						alt={item.title}
						class="max-w-48 mx-auto aspect-[2/3] object-cover rounded-lg"
					/>
				{/if}
				{#if item.overview}
					<p class="text-sm pt-2">{item.overview}</p>
				{/if}
			</div>
			<Dialog.Footer>
				<Button onclick={handleViewDetails}>More details</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>

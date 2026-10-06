<script lang="ts">
	import { goto } from '$app/navigation';
	import type { MediaItem } from '$lib/api/media';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import MoviePoster from '$lib/components/boxoffice/movie-poster.svelte';
	import { createRequest } from '$lib/api/seerr';
	import { retrySuggestions, blacklistSuggestions } from '$lib/api/suggestarr';

	const AVAILABILITY_LABELS: Record<number, string> = {
		1: 'Unknown',
		2: 'Pending',
		3: 'Processing',
		4: 'Partially Available',
		5: 'Available',
		6: 'Deleted',
	};

	let {
		open = false,
		item = null,
		onclose,
		onRequested,
	}: { open?: boolean; item?: MediaItem | null; onclose?: () => void; onRequested?: () => void } = $props();

	let requesting = $state(false);
	let requestError: string | null = $state(null);
	let requested = $state(false);

	$effect(() => {
		if (open) {
			requesting = false;
			requestError = null;
			requested = false;
		}
	});

	function handleOpenChange(next: boolean) {
		if (!next) onclose?.();
	}

	function formatMoney(thousands?: number) {
		if (!thousands) return '$0';
		return `$${(thousands / 1000).toFixed(1)}M`;
	}

	async function handleRequest() {
		if (!item?.tmdbId) return;
		requesting = true;
		requestError = null;
		try {
			await createRequest({
				mediaType: item.mediaType,
				mediaId: item.tmdbId,
				...(item.mediaType === 'tv' ? { seasons: 'all' } : {}),
			});
			requested = true;
			onRequested?.();
		} catch (e) {
			requestError = (e as Error).message;
		} finally {
			requesting = false;
		}
	}

	function handleViewDetails() {
		if (!item?.tmdbId) return;
		const url = `/media/${item.mediaType}/${item.tmdbId}`;
		onclose?.();
		goto(url);
	}

	function handleApprove() {
		if (item?.source !== 'suggestarr') return;
		retrySuggestions([(item.raw as any).id]);
		onclose?.();
	}

	function handleReject() {
		if (item?.source !== 'suggestarr') return;
		blacklistSuggestions([(item.raw as any).id]);
		onclose?.();
	}

	let alreadyAvailable = $derived(item?.availabilityStatus === 5 || item?.availabilityStatus === 4);
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
	<Dialog.Content>
		{#if item}
			<Dialog.Header>
				<div class="flex items-center gap-2">
					<Badge variant="secondary">{item.mediaType === 'tv' ? 'TV' : 'Movie'}</Badge>
					{#if item.rating}
						<span class="text-xs text-muted-foreground">★ {item.rating.toFixed(1)}</span>
					{/if}
					{#if item.releaseDate}
						<span class="text-xs text-muted-foreground">{item.releaseDate}</span>
					{/if}
				</div>
				<Dialog.Title>{item.title}</Dialog.Title>
			</Dialog.Header>
			<div class="grid gap-2">
				{#if item.source === 'boxarr'}
					<MoviePoster radarrId={(item.raw as any).radarr_id} title={item.title} class="max-w-48 mx-auto rounded-lg" />
				{:else if item.posterUrl}
					<img src={item.posterUrl} alt={item.title} class="max-w-48 mx-auto aspect-[2/3] object-cover rounded-lg" />
				{/if}

				{#if item.availabilityStatus}
					<Badge variant="outline" class="w-fit">{AVAILABILITY_LABELS[item.availabilityStatus] || 'Unknown'}</Badge>
				{/if}

				{#if item.source === 'boxarr'}
					{@const m = item.raw as any}
					<div class="flex flex-wrap gap-1.5">
						{#if m.radarr_has_file}
							<Badge variant="default">Available in Radarr</Badge>
						{:else if m.is_new_release}
							<Badge variant="secondary">New Release</Badge>
						{/if}
						{#if m.radarr_status}
							<Badge variant="outline">{m.radarr_status}</Badge>
						{/if}
					</div>
					<p class="text-sm text-muted-foreground">Weekend Gross: <span class="text-foreground font-medium">{formatMoney(m.weekend_gross)}</span></p>
					<p class="text-sm text-muted-foreground">Total Gross: <span class="text-foreground font-medium">{formatMoney(m.total_gross)}</span></p>
					{#if m.weeks_in_release}
						<p class="text-sm text-muted-foreground">Weeks in Release: {m.weeks_in_release}</p>
					{/if}
				{/if}

				{#if item.source === 'suggestarr'}
					<Badge variant="outline" class="w-fit">{(item.raw as any).status}</Badge>
				{/if}

				{#if item.overview}
					<p class="text-sm pt-2">{item.overview}</p>
				{/if}

				{#if requestError}
					<p class="text-sm text-destructive">{requestError}</p>
				{/if}
			</div>
			<Dialog.Footer>
				{#if item.source === 'suggestarr'}
					<Button variant="outline" onclick={handleReject}>Blacklist</Button>
					<Button onclick={handleApprove}>Approve</Button>
				{/if}
				{#if item.tmdbId}
					<Button variant="outline" onclick={handleViewDetails}>More details</Button>
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
				{:else if item.source === 'boxarr'}
					<span class="text-xs text-muted-foreground self-center">Already tracked in Radarr</span>
				{/if}
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>

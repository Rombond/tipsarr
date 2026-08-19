<script lang="ts">
	import { posterUrl, getMovieDetails, getTvDetails } from '$lib/api/seerr';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';

	// Seerr MediaRequest shape (subset)
	let {
		request,
		isAdmin = false,
		onApprove,
		onDecline,
		onRetry,
		onDelete,
	}: {
		request: any;
		isAdmin?: boolean;
		onApprove?: (id: number) => void;
		onDecline?: (id: number) => void;
		onRetry?: (id: number) => void;
		onDelete?: (id: number) => void;
	} = $props();

	const STATUS_LABELS: Record<number, string> = {
		1: 'Pending',
		2: 'Approved',
		3: 'Declined',
	};

	let media = $derived(request.media);
	let details: any = $state(null);

	$effect(() => {
		const tmdbId = media?.tmdbId;
		details = null;
		if (!tmdbId) return;
		const mediaType = media?.mediaType === 'tv' ? 'tv' : media?.mediaType === 'movie' ? 'movie' : null;
		const fetchers = mediaType === 'tv' ? [getTvDetails] : mediaType === 'movie' ? [getMovieDetails] : [getMovieDetails, getTvDetails];
		(async () => {
			for (const fetcher of fetchers) {
				try {
					details = await fetcher(tmdbId);
					return;
				} catch {
					// try next
				}
			}
		})();
	});

	let title = $derived(details?.title || details?.name || `Request #${request.id}`);
	let poster = $derived(posterUrl(details?.posterPath));
</script>

<div class="flex items-center gap-3 border-b border-border py-2">
	<div class="h-16 w-11 shrink-0 overflow-hidden rounded bg-muted">
		{#if poster}
			<img src={poster} alt={title} class="h-full w-full object-cover" loading="lazy" />
		{/if}
	</div>
	<div class="min-w-0 flex-1">
		<p class="truncate font-medium text-sm">{title}</p>
		<div class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
			<Badge variant="outline" class="text-[10px]">{STATUS_LABELS[request.status] || 'Unknown'}</Badge>
			<span>{request.requestedBy?.displayName || request.requestedBy?.username || 'Unknown user'}</span>
			<span>· requested {new Date(request.createdAt).toLocaleDateString()}</span>
			<span>· updated {new Date(request.updatedAt).toLocaleDateString()}</span>
		</div>
	</div>
	<div class="flex shrink-0 gap-1.5">
		{#if isAdmin && request.status === 1}
			<Button size="sm" variant="outline" onclick={() => onDecline?.(request.id)}>Decline</Button>
			<Button size="sm" onclick={() => onApprove?.(request.id)}>Approve</Button>
		{/if}
		<Button size="sm" variant="outline" onclick={() => onRetry?.(request.id)}>Retry</Button>
		<Button size="sm" variant="outline" onclick={() => onDelete?.(request.id)}>Cancel</Button>
	</div>
</div>

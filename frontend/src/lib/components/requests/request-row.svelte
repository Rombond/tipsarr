<script lang="ts">
	import { imageUrl, type Schemas } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';

	let {
		request,
		isAdmin,
		busy = false,
		onApprove,
		onDecline,
		onDelete,
	}: {
		request: Schemas['View'];
		isAdmin: boolean;
		busy?: boolean;
		onApprove?: (r: Schemas['View']) => void;
		onDecline?: (r: Schemas['View']) => void;
		onDelete?: (r: Schemas['View']) => void;
	} = $props();

	const poster = $derived(imageUrl(request.posterPath, 'w154'));
	const mins = (s: number) => (s >= 3600 ? `${Math.floor(s / 3600)} h ${Math.round((s % 3600) / 60)} min` : `${Math.max(1, Math.round(s / 60))} min`);

	const stageLabel = $derived(
		{
			requested: 'Pending approval',
			approved: request.dryRun ? 'Approved (dry-run: not sent)' : 'Approved',
			searching: 'Searching',
			downloading: request.progress
				? `Downloading ${request.progress.percent}%${request.progress.etaSeconds ? ` · ${mins(request.progress.etaSeconds)} left` : ''}`
				: 'Downloading',
			available: 'Available',
			declined: 'Declined',
			failed: 'Failed',
		}[request.stage],
	);
	const variant = $derived(
		request.stage === 'failed' || request.stage === 'declined' ? 'destructive' : request.stage === 'available' ? 'default' : 'secondary',
	);
	const canDelete = $derived(isAdmin || ['pending', 'declined', 'failed'].includes(request.status));
</script>

<div class="flex gap-3 rounded-lg border border-border p-3">
	<a href="/media/{request.type}/{request.tmdbId}" class="w-16 shrink-0" aria-label={request.title}>
		{#if poster}
			<img src={poster} alt={request.title} class="aspect-[2/3] w-full rounded-md object-cover" loading="lazy" />
		{:else}
			<div class="aspect-[2/3] w-full rounded-md bg-muted"></div>
		{/if}
	</a>
	<div class="grid min-w-0 flex-1 content-start gap-1.5">
		<div class="flex flex-wrap items-center gap-2">
			<a href="/media/{request.type}/{request.tmdbId}" class="truncate font-medium hover:underline">{request.title}</a>
			<Badge variant="outline" class="text-[10px]">{request.type === 'tv' ? 'TV' : 'Movie'}</Badge>
			<Badge {variant}>{stageLabel}</Badge>
		</div>
		<p class="text-xs text-muted-foreground">
			Requested by {request.requestedBy.name || 'unknown'} · {new Date(request.createdAt * 1000).toLocaleDateString()}
			{#if request.seasons?.length}· Season{request.seasons.length > 1 ? 's' : ''} {request.seasons.join(', ')}{/if}
			{#if request.decidedBy}· decided by {request.decidedBy.name}{/if}
		</p>
		{#if request.stage === 'downloading' && request.progress}
			<div class="h-1.5 w-full max-w-sm overflow-hidden rounded-full bg-muted">
				<div class="h-full bg-primary transition-all" style="width: {request.progress.percent}%"></div>
			</div>
		{/if}
		{#if request.declineReason}<p class="text-xs">Reason: {request.declineReason}</p>{/if}
		{#if request.error}<p class="text-xs text-destructive">{request.error}</p>{/if}
		{#if isAdmin || canDelete}
			<div class="flex flex-wrap gap-2 pt-1">
				{#if isAdmin && (request.status === 'pending' || request.status === 'failed')}
					<Button size="sm" disabled={busy} onclick={() => onApprove?.(request)}>
						{request.status === 'failed' ? 'Retry' : 'Approve'}
					</Button>
					<Button size="sm" variant="outline" disabled={busy} onclick={() => onDecline?.(request)}>Decline</Button>
				{/if}
				{#if canDelete}
					<Button size="sm" variant="ghost" disabled={busy} onclick={() => onDelete?.(request)}>Delete</Button>
				{/if}
			</div>
		{/if}
	</div>
</div>

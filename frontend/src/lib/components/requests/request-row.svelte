<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate, fmtDuration } from '$lib/i18n/format';
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

	const stageLabel = $derived(
		{
			requested: t('req.stage.requested'),
			approved: request.dryRun ? t('req.stage.approved_dry') : t('req.stage.approved'),
			searching: t('req.stage.searching'),
			downloading: request.progress
				? request.progress.etaSeconds
					? t('req.stage.downloading_eta', { percent: request.progress.percent, eta: fmtDuration(request.progress.etaSeconds, t) })
					: t('req.stage.downloading_pct', { percent: request.progress.percent })
				: t('req.stage.downloading'),
			available: t('req.stage.available'),
			declined: t('req.stage.declined'),
			failed: t('req.stage.failed'),
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
			<Badge variant="outline" class="text-[10px]">{request.type === 'tv' ? t('type.tv_short') : t('type.movie')}</Badge>
			<Badge {variant}>{stageLabel}</Badge>
		</div>
		<p class="text-xs text-muted-foreground">
			{t('req.requested_by', { name: request.requestedBy.name || t('req.unknown_user') })} · {fmtDate(request.createdAt)}
			{#if request.seasons?.length}· {t('req.seasons', { count: request.seasons.length, list: request.seasons.join(', ') })}{/if}
			{#if request.decidedBy}· {t('req.decided_by', { name: request.decidedBy.name })}{/if}
		</p>
		{#if request.stage === 'downloading' && request.progress}
			<div class="h-1.5 w-full max-w-sm overflow-hidden rounded-full bg-muted">
				<div class="h-full bg-primary transition-all" style="width: {request.progress.percent}%"></div>
			</div>
		{/if}
		{#if request.declineReason}<p class="text-xs">{t('req.reason', { reason: request.declineReason })}</p>{/if}
		{#if request.error}<p class="text-xs text-destructive">{request.error}</p>{/if}
		{#if isAdmin || canDelete}
			<div class="flex flex-wrap gap-2 pt-1">
				{#if isAdmin && (request.status === 'pending' || request.status === 'failed')}
					<Button size="sm" disabled={busy} onclick={() => onApprove?.(request)}>
						{request.status === 'failed' ? t('req.retry') : t('req.approve')}
					</Button>
					<Button size="sm" variant="outline" disabled={busy} onclick={() => onDecline?.(request)}>{t('req.decline')}</Button>
				{/if}
				{#if canDelete}
					<Button size="sm" variant="ghost" disabled={busy} onclick={() => onDelete?.(request)}>{t('req.delete')}</Button>
				{/if}
			</div>
		{/if}
	</div>
</div>

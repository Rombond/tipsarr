<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate, fmtDuration } from '$lib/i18n/format';
	import { imageUrl, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import StatusBadge from '$lib/components/ui/status-badge.svelte';
	import { Button } from '$lib/components/ui/button';
	import type { StatusKey } from '$lib/status';
	import XIcon from '@lucide/svelte/icons/x';
	import SlidersIcon from '@lucide/svelte/icons/sliders-horizontal';
	import RequestOptionsDialog from './request-options-dialog.svelte';

	let {
		request,
		isAdmin,
		busy = false,
		onApprove,
		onDecline,
		onDelete,
		onChanged,
	}: {
		request: Schemas['View'];
		isAdmin: boolean;
		busy?: boolean;
		onApprove?: (r: Schemas['View']) => void;
		onDecline?: (r: Schemas['View']) => void;
		onDelete?: (r: Schemas['View']) => void;
		/** Called after the quality profile was changed. */
		onChanged?: () => void;
	} = $props();

	let editing = $state(false);

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
	const canDelete = $derived(isAdmin || ['pending', 'declined', 'failed'].includes(request.status));
	// the requester and admins may change the quality profile until it is sent
	const canEdit = $derived(!request.source && (request.status === 'pending' || request.status === 'failed') && (isAdmin || request.requestedBy.id === auth.user?.id));
	const canDecide = $derived(isAdmin && (request.status === 'pending' || request.status === 'failed'));

	// a profile opens for yourself, or for any user when you are an admin
	const canOpen = (id: string) => !!id && (auth.isAdmin || id === auth.user?.id);
</script>

{#snippet person(u: { id: string; name: string })}
	{@const name = u.name || t('req.unknown_user')}
	{#if canOpen(u.id)}<a href="/users/{u.id}" class="font-medium text-foreground hover:underline">{name}</a>{:else}<span class="font-medium text-foreground">{name}</span>{/if}
{/snippet}

<div class="relative flex gap-3 rounded-xl border border-border bg-card p-3 shadow-sm">
	{#if canEdit}
		<button
			type="button"
			class="absolute top-2 right-10 flex size-7 cursor-pointer items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:opacity-50"
			title={t('req.change_options_short')}
			aria-label={t('req.change_options', { title: request.title })}
			disabled={busy}
			onclick={() => (editing = true)}
		>
			<SlidersIcon class="size-4" />
		</button>
	{/if}
	{#if canDelete}
		<button
			type="button"
			class="absolute top-2 right-2 flex size-7 cursor-pointer items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-rose-500/15 hover:text-rose-600 disabled:opacity-50 dark:hover:text-rose-400"
			title={t('req.delete')}
			aria-label={t('req.delete_aria', { title: request.title })}
			disabled={busy}
			onclick={() => onDelete?.(request)}
		>
			<XIcon class="size-4" />
		</button>
	{/if}
	<a href="/media/{request.type}/{request.tmdbId}" class="w-16 shrink-0 sm:w-20" aria-label={request.title}>
		{#if poster}
			<img src={poster} alt="" class="aspect-[2/3] w-full rounded-md object-cover" loading="lazy" />
		{:else}
			<div class="aspect-[2/3] w-full rounded-md bg-muted"></div>
		{/if}
	</a>
	<div class="grid min-w-0 flex-1 content-start gap-1.5 pr-6">
		<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
			<a href="/media/{request.type}/{request.tmdbId}" class="truncate font-semibold hover:underline">{request.title}</a>
			<span class="rounded-full bg-violet-500/15 px-2 py-0.5 text-[10px] font-semibold tracking-wide text-violet-700 uppercase dark:text-violet-300">
				{request.type === 'tv' ? t('type.tv_short') : t('type.movie')}
			</span>
		</div>
		<div><StatusBadge status={request.stage as StatusKey}>{stageLabel}</StatusBadge></div>
		<p class="text-xs text-muted-foreground">
			{#if request.source}
				{t('req.imported_from', { source: request.source === 'sonarr' ? 'Sonarr' : 'Radarr' })}
			{:else}
				{t('req.requested_by_link')} {@render person(request.requestedBy)}
			{/if}
			· {fmtDate(request.createdAt)}
			{#if request.decidedBy}· {t('req.decided_by_link')} {@render person(request.decidedBy)}{/if}
		</p>
		{#if request.seasons?.length}
			<div class="flex flex-wrap items-center gap-1" aria-label={t('req.seasons', { count: request.seasons.length, list: request.seasons.join(', ') })}>
				{#each request.seasons as n (n)}
					<span class="rounded-md bg-sky-500/15 px-1.5 py-0.5 text-[11px] font-semibold text-sky-700 dark:text-sky-300">{t('req.season_badge', { n })}</span>
				{/each}
			</div>
		{/if}
		{#if request.stage === 'downloading' && request.progress}
			<div class="h-1.5 w-full max-w-sm overflow-hidden rounded-full bg-muted">
				<div class="h-full bg-indigo-500 transition-all" style="width: {request.progress.percent}%"></div>
			</div>
		{/if}
		{#if request.declineReason}<p class="text-xs">{t('req.reason', { reason: request.declineReason })}</p>{/if}
		{#if request.error}<p class="text-xs text-destructive">{request.error}</p>{/if}
		{#if canDecide}
			<div class="flex flex-wrap justify-end gap-2 pt-1">
				<Button size="sm" class="bg-rose-600 text-white hover:bg-rose-600/90" disabled={busy} onclick={() => onDecline?.(request)}>{t('req.decline')}</Button>
				<Button size="sm" class="bg-emerald-600 text-white hover:bg-emerald-600/90" disabled={busy} onclick={() => onApprove?.(request)}>
					{request.status === 'failed' ? t('req.retry') : t('req.approve')}
				</Button>
			</div>
		{/if}
	</div>
</div>

{#if canEdit}<RequestOptionsDialog bind:open={editing} {request} onsaved={() => onChanged?.()} />{/if}

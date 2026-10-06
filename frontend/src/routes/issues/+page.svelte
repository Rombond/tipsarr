<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { api, unwrap, imageUrl, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { onEvent, stream } from '$lib/events.svelte';
	import StatusBadge from '$lib/components/ui/status-badge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MessageIcon from '@lucide/svelte/icons/message-square';

	type Filter = 'open' | 'resolved' | 'all';
	const filters: Filter[] = ['open', 'resolved', 'all'];

	let filter = $state<Filter>('open');
	let items = $state<Schemas['IssueView'][]>([]);
	let total = $state(0);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load(spinner = true) {
		if (spinner) loading = true;
		try {
			const res = await unwrap(api.GET('/issues', { params: { query: { filter, take: 50 } } }));
			items = res.items;
			total = res.total;
			error = null;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		filter;
		stream.reconnects;
		load();
	});
	$effect(() => onEvent('issue.updated', () => load(false)));

	const scope = (i: Schemas['IssueView']) => (i.type === 'tv' && i.season ? (i.episode ? t('issue.s_e', { s: i.season, e: i.episode }) : t('req.season_badge', { n: i.season })) : '');
</script>

<svelte:head>
	<title>{t('issue.title')} · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
	<div>
		<h1 class="font-bold text-3xl">{t('issue.title')}</h1>
		<p class="text-sm text-muted-foreground">{auth.isAdmin ? t('issue.subtitle_admin') : t('issue.subtitle')}</p>
	</div>

	<div class="flex gap-1" role="tablist" aria-label={t('issue.title')}>
		{#each filters as f (f)}
			<button
				type="button"
				role="tab"
				aria-selected={filter === f}
				class="cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors {filter === f ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
				onclick={() => (filter = f)}
			>{t(`issue.filter.${f}` as 'issue.filter.open')}</button>
		{/each}
	</div>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading}
		<Skeleton class="h-20 w-full" />
		<Skeleton class="h-20 w-full" />
	{:else if items.length === 0}
		<div class="rounded-xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">{t('issue.empty')}</div>
	{:else}
		<div class="grid gap-3 xl:grid-cols-2">
			{#each items as i (i.id)}
				{@const poster = imageUrl(i.posterPath, 'w154')}
				<a href="/issues/{i.id}" class="flex gap-3 rounded-xl border border-border bg-card p-3 shadow-sm transition-colors hover:border-primary/50">
					{#if poster}<img src={poster} alt="" class="aspect-[2/3] w-14 shrink-0 rounded-md object-cover" loading="lazy" />{:else}<div class="aspect-[2/3] w-14 shrink-0 rounded-md bg-muted"></div>{/if}
					<div class="grid min-w-0 flex-1 content-start gap-1.5">
						<p class="truncate font-semibold">{i.title}{#if scope(i)} <span class="font-normal text-muted-foreground">· {scope(i)}</span>{/if}</p>
						<div class="flex flex-wrap items-center gap-1.5">
							<StatusBadge status={i.status === 'open' ? 'failed' : 'available'}>{i.status === 'open' ? t('issue.open') : t('issue.resolved')}</StatusBadge>
							<span class="rounded-full bg-violet-500/15 px-2 py-0.5 text-xs font-semibold text-violet-700 dark:text-violet-300">{t(`issue.kind.${i.kind}` as 'issue.kind.video')}</span>
						</div>
						<p class="flex flex-wrap items-center gap-x-2 text-xs text-muted-foreground">
							<span>{t('issue.reported_by', { name: i.createdBy.name || t('req.unknown_user') })}</span>
							<span>· {fmtDate(i.createdAt)}</span>
							<span class="inline-flex items-center gap-1"><MessageIcon class="size-3" />{i.commentCount}</span>
						</p>
					</div>
				</a>
			{/each}
		</div>
		<p class="text-xs text-muted-foreground">{t('issue.total', { count: total })}</p>
	{/if}
</div>

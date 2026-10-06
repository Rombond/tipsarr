<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDateTime } from '$lib/i18n/format';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, imageUrl, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { onEvent } from '$lib/events.svelte';
	import { toast } from '$lib/toast.svelte';
	import StatusBadge from '$lib/components/ui/status-badge.svelte';
	import UserAvatar from '$lib/components/users/user-avatar.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	const id = $derived(page.params.id ?? '');
	let issue = $state<Schemas['IssueThread'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reply = $state('');
	let busy = $state(false);

	async function load(spinner = true) {
		if (spinner) loading = true;
		try {
			issue = await unwrap(api.GET('/issues/{id}', { params: { path: { id } } }));
			error = null;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		id;
		load();
	});
	$effect(() =>
		onEvent('issue.updated', (d: { id: string }) => {
			if (d.id === id) load(false);
		}),
	);

	async function act(fn: () => Promise<unknown>) {
		busy = true;
		try {
			await fn();
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			busy = false;
		}
	}

	const send = (e: SubmitEvent) => {
		e.preventDefault();
		if (!reply.trim()) return;
		act(async () => {
			issue = await unwrap(api.POST('/issues/{id}/comments', { params: { path: { id } }, body: { message: reply.trim() } }));
			reply = '';
		});
	};
	const setResolved = (resolved: boolean) =>
		act(async () => {
			issue = await unwrap(api.POST(resolved ? '/issues/{id}/resolve' : '/issues/{id}/reopen', { params: { path: { id } } }));
		});
	const remove = () => {
		if (!confirm(t('issue.confirm_delete'))) return;
		act(async () => {
			await expectOk(api.DELETE('/issues/{id}', { params: { path: { id } } }));
			goto('/issues');
		});
	};

	const canOpen = (uid: string) => !!uid && (auth.isAdmin || uid === auth.user?.id);
</script>

<svelte:head>
	<title>{issue?.title ?? t('issue.title')} · Tipsarr</title>
</svelte:head>

<div class="grid max-w-3xl grid-cols-[minmax(0,1fr)] gap-5">
	<a href="/issues" class="inline-flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"><ArrowLeftIcon class="size-4" />{t('issue.back')}</a>

	{#if loading}
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-40 w-full" />
	{:else if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if issue}
		{@const poster = imageUrl(issue.posterPath, 'w154')}
		<header class="flex gap-4">
			{#if poster}<a href="/media/{issue.type}/{issue.tmdbId}"><img src={poster} alt="" class="aspect-[2/3] w-20 rounded-md object-cover shadow" /></a>{/if}
			<div class="grid content-start gap-1.5">
				<h1 class="font-bold text-2xl"><a href="/media/{issue.type}/{issue.tmdbId}" class="hover:underline">{issue.title}</a></h1>
				<div class="flex flex-wrap items-center gap-1.5">
					<StatusBadge status={issue.status === 'open' ? 'failed' : 'available'}>{issue.status === 'open' ? t('issue.open') : t('issue.resolved')}</StatusBadge>
					<span class="rounded-full bg-violet-500/15 px-2 py-0.5 text-xs font-semibold text-violet-700 dark:text-violet-300">{t(`issue.kind.${issue.kind}` as 'issue.kind.video')}</span>
					{#if issue.type === 'tv' && issue.season}
						<span class="rounded-full bg-sky-500/15 px-2 py-0.5 text-xs font-semibold text-sky-700 dark:text-sky-300">
							{issue.episode ? t('issue.s_e', { s: issue.season, e: issue.episode }) : t('req.season_badge', { n: issue.season })}
						</span>
					{/if}
				</div>
				<p class="text-xs text-muted-foreground">
					{t('issue.reported_by_link')}
					{#if canOpen(issue.createdBy.id)}<a class="font-medium text-foreground hover:underline" href="/users/{issue.createdBy.id}">{issue.createdBy.name}</a>{:else}<span class="font-medium text-foreground">{issue.createdBy.name}</span>{/if}
					· {fmtDateTime(issue.createdAt)}
					{#if issue.resolvedBy}· {t('issue.resolved_by', { name: issue.resolvedBy.name })}{/if}
				</p>
			</div>
		</header>

		<section class="grid gap-3" aria-label={t('issue.thread')}>
			{#each issue.comments as c (c.id)}
				<div class="flex gap-3">
					<UserAvatar id={c.user.id} name={c.user.name} class="size-9 shrink-0" />
					<div class="min-w-0 flex-1 rounded-xl border border-border bg-card px-4 py-3">
						<p class="mb-1 flex flex-wrap items-baseline gap-x-2 text-xs text-muted-foreground">
							<span class="font-semibold text-foreground">{c.user.name}</span>{fmtDateTime(c.createdAt)}
						</p>
						<p class="text-sm leading-6 whitespace-pre-wrap">{c.message}</p>
					</div>
				</div>
			{/each}
		</section>

		<form class="grid gap-2" onsubmit={send}>
			<textarea
				bind:value={reply}
				rows="3"
				maxlength="2000"
				placeholder={t('issue.reply_placeholder')}
				aria-label={t('issue.reply_placeholder')}
				class="rounded-md border border-input bg-background px-3 py-2 text-sm shadow-xs outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
			></textarea>
			<div class="flex flex-wrap justify-end gap-2">
				{#if auth.isAdmin}<Button type="button" variant="ghost" class="mr-auto text-destructive" disabled={busy} onclick={remove}>{t('req.delete')}</Button>{/if}
				{#if issue.status === 'open'}
					<Button type="button" variant="outline" disabled={busy} onclick={() => setResolved(true)}>{t('issue.resolve')}</Button>
				{:else}
					<Button type="button" variant="outline" disabled={busy} onclick={() => setResolved(false)}>{t('issue.reopen')}</Button>
				{/if}
				<Button type="submit" disabled={busy || !reply.trim()}>{t('issue.comment')}</Button>
			</div>
		</form>
	{/if}
</div>

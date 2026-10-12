<script lang="ts">
	// Titles one person swiped past in Teasers; an admin can make them show up again.
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { api, unwrap, expectOk, errorText, imageUrl, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';

	type Row = Schemas['HistoryItem'];

	let {
		open = $bindable(false),
		user,
	}: {
		open?: boolean;
		user: Schemas['User'];
	} = $props();

	const PAGE = 40;
	let items = $state<Row[]>([]);
	let total = $state(0);
	let loading = $state(false);
	let exhausted = $state(false);
	let error = $state<string | null>(null);
	let loadedFor = '';

	async function loadMore(reset = false) {
		if (loading) return;
		loading = true;
		error = null;
		const id = user.id;
		try {
			const offset = reset ? 0 : items.length;
			const r = await unwrap(api.GET('/admin/users/{id}/teasers/history', { params: { path: { id }, query: { limit: PAGE, offset } } }));
			if (id !== user.id) return;
			items = reset ? r.items : [...items, ...r.items];
			total = r.total;
			// a page can hold fewer titles than asked for; an empty one means nothing more is known
			exhausted = r.items.length === 0;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (!open) {
			loadedFor = '';
			return;
		}
		if (loadedFor === user.id) return;
		loadedFor = user.id;
		items = [];
		total = 0;
		exhausted = false;
		loadMore(true);
	});

	async function forget(row: Row) {
		error = null;
		try {
			await expectOk(api.DELETE('/admin/users/{id}/teasers/seen/{type}/{tmdbId}', { params: { path: { id: user.id, type: row.type, tmdbId: row.tmdbId } } }));
			items = items.filter((i) => !(i.type === row.type && i.tmdbId === row.tmdbId));
			total = Math.max(0, total - 1);
			toast.success(t('users.teasers_history_forgot', { title: row.title }));
		} catch (e) {
			error = errorText(e);
		}
	}

	async function forgetAll() {
		if (!confirm(t('users.teasers_history_confirm_all', { name: user.name }))) return;
		error = null;
		try {
			await expectOk(api.DELETE('/admin/users/{id}/teasers/seen', { params: { path: { id: user.id } } }));
			items = [];
			total = 0;
			toast.success(t('users.teasers_history_wiped'));
		} catch (e) {
			error = errorText(e);
		}
	}

	const year = (r: Row) => r.releaseDate?.slice(0, 4) ?? '';
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-xl">
		<Dialog.Header>
			<Dialog.Title>{t('users.teasers_history_title', { name: user.name })}</Dialog.Title>
			<Dialog.Description>{t('users.teasers_history_hint', { name: user.name })}</Dialog.Description>
		</Dialog.Header>
		{#if items.length}
			<div class="flex justify-end">
				<Button variant="outline" size="sm" onclick={forgetAll}>{t('users.teasers_history_forget_all')}</Button>
			</div>
		{/if}
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<div class="grid gap-2">
			{#each items as row (`${row.type}:${row.tmdbId}`)}
				{@const poster = imageUrl(row.posterPath, 'w92')}
				<div class="flex items-center gap-3 rounded-md border border-border p-2 text-sm">
					{#if poster}
						<img src={poster} alt="" loading="lazy" class="h-14 w-10 shrink-0 rounded object-cover" />
					{:else}
						<div class="h-14 w-10 shrink-0 rounded bg-muted"></div>
					{/if}
					<div class="min-w-0 flex-1">
						<div class="truncate font-medium">{row.title}{#if year(row)} <span class="font-normal text-muted-foreground">({year(row)})</span>{/if}</div>
						<div class="text-xs text-muted-foreground">{t('users.teasers_history_seen', { date: fmtDate(row.seenAt) })}</div>
					</div>
					<Button variant="ghost" size="sm" onclick={() => forget(row)}>{t('users.teasers_history_forget')}</Button>
				</div>
			{:else}
				{#if !loading}<p class="text-sm text-muted-foreground">{t('users.teasers_history_empty')}</p>{/if}
			{/each}
		</div>
		{#if items.length && items.length < total && !exhausted}
			<div class="flex justify-center">
				<Button variant="outline" size="sm" disabled={loading} onclick={() => loadMore()}>{t('users.teasers_history_more')}</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

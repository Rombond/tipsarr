<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { onEvent, stream } from '$lib/events.svelte';
	import Carousel from '$lib/components/media/carousel.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let { onSelect }: { onSelect?: (item: Schemas['Item']) => void } = $props();

	let rows = $state<Schemas['Row'][]>([]);
	let generating = $state(false);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let refreshing = $state(false);
	let refreshMessage = $state<string | null>(null);

	async function load(spinner = true) {
		if (spinner) loading = true;
		try {
			const res = await unwrap(api.GET('/suggestions'));
			rows = res.rows;
			generating = res.generating;
			error = null;
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	// first load, and again after an SSE reconnect
	$effect(() => {
		stream.reconnects;
		load();
	});

	$effect(() =>
		onEvent('suggestions.updated', () => {
			// only react to updates we are waiting for (never reload on unsolicited events)
			if (!generating && !refreshing) return;
			refreshing = false;
			load(false);
		}),
	);

	function rowTitle(row: Schemas['Row']): string {
		if (row.variant === 'because') return t('suggest.because', { title: row.seed?.title ?? '' });
		return t(`suggest.${row.variant}` as 'suggest.personal');
	}

	async function dismiss(item: Schemas['Item']) {
		// optimistic: remove it everywhere right away, restore on failure
		const before = rows;
		rows = rows.map((r) => ({ ...r, items: r.items.filter((i) => !(i.type === item.type && i.tmdbId === item.tmdbId)) }));
		try {
			await expectOk(api.POST('/blocklist', { body: { type: item.type, tmdbId: item.tmdbId } }));
		} catch {
			rows = before;
		}
	}

	async function refresh() {
		refreshing = true;
		refreshMessage = null;
		const { response } = await api.POST('/suggestions/refresh');
		if (response.status === 429) {
			refreshMessage = t('error.refresh_too_soon');
			refreshing = false;
		} else if (!response.ok) {
			refreshMessage = t('suggest.refresh_failed');
			refreshing = false;
		}
		// on success the suggestions.updated event stops the spinner
	}
</script>

{#if loading}
	<div class="grid gap-2">
		<Skeleton class="h-5 w-48" />
		<Skeleton class="h-56 w-full" />
		<p class="text-xs text-muted-foreground">{t('suggest.preparing')}</p>
	</div>
{:else if error}
	<p class="text-sm text-muted-foreground">{t('suggest.unavailable', { error })}</p>
{:else}
	<div class="grid gap-6">
		{#each rows as row (row.id)}
			<Carousel title={rowTitle(row)} items={row.items} {onSelect} onDismiss={dismiss} />
		{/each}
		<div class="-mt-3 flex items-center justify-end gap-3 text-xs text-muted-foreground">
			{#if refreshMessage}<span>{refreshMessage}</span>{/if}
			<Button size="sm" variant="ghost" class="h-7 px-2 text-xs" disabled={refreshing || generating} onclick={refresh}>
				{refreshing || generating ? t('suggest.updating') : t('suggest.refresh')}
			</Button>
		</div>
	</div>
{/if}

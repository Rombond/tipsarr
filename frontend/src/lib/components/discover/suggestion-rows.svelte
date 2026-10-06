<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { onEvent, stream } from '$lib/events.svelte';
	import Carousel from '$lib/components/media/carousel.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';

	import type { Snippet } from 'svelte';

	let { onSelect, fillers = [] }: { onSelect?: (item: Schemas['Item']) => void; /** Other sections to weave between the "Because you watched" rows. */ fillers?: Snippet[] } = $props();

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

	// "Because you watched X" rows would otherwise sit in one block: alternate them with the other sections
	const sequence = $derived.by(() => {
		const lead = rows.filter((r) => r.variant !== 'because');
		const because = rows.filter((r) => r.variant === 'because');
		const out: ({ row: Schemas['Row'] } | { filler: Snippet })[] = lead.map((row) => ({ row }));
		const spare = [...fillers];
		for (const row of because) {
			const f = spare.shift();
			if (f && out.length) out.push({ filler: f });
			else if (f) spare.unshift(f);
			out.push({ row });
		}
		return [...out, ...spare.map((filler) => ({ filler }))];
	});

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
	{#each fillers as filler}{@render filler()}{/each}
{:else if error}
	<p class="text-sm text-muted-foreground">{t('suggest.unavailable', { error })}</p>
	{#each fillers as filler}{@render filler()}{/each}
{:else}
	<div class="grid gap-8">
		{#each sequence as part, i (i)}
			{#if 'row' in part}
				<Carousel title={rowTitle(part.row)} items={part.row.items} {onSelect} onDismiss={dismiss} />
			{:else}
				{@render part.filler()}
			{/if}
		{/each}
		<div class="-mt-3 flex items-center justify-end gap-3 text-xs text-muted-foreground">
			{#if refreshMessage}<span>{refreshMessage}</span>{/if}
			<Button size="sm" variant="ghost" class="h-7 px-2 text-xs" disabled={refreshing || generating} onclick={refresh}>
				{refreshing || generating ? t('suggest.updating') : t('suggest.refresh')}
			</Button>
		</div>
	</div>
{/if}

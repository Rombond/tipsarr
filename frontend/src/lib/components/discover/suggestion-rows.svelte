<script lang="ts">
	import { api, unwrap, type Schemas } from '$lib/api/client';
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
			error = (e as Error).message;
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
			refreshing = false;
			load(false);
		}),
	);

	async function refresh() {
		refreshing = true;
		refreshMessage = null;
		const { response } = await api.POST('/suggestions/refresh');
		if (response.status === 429) {
			refreshMessage = 'Refreshed a moment ago, try again in a minute.';
			refreshing = false;
		} else if (!response.ok) {
			refreshMessage = 'Could not refresh right now.';
			refreshing = false;
		}
		// on success the suggestions.updated event stops the spinner
	}
</script>

{#if loading}
	<div class="grid gap-2">
		<Skeleton class="h-5 w-48" />
		<Skeleton class="h-56 w-full" />
		<p class="text-xs text-muted-foreground">Preparing your recommendations…</p>
	</div>
{:else if error}
	<p class="text-sm text-muted-foreground">Recommendations unavailable: {error}</p>
{:else}
	<div class="grid gap-6">
		{#each rows as row (row.id)}
			<Carousel title={row.title} items={row.items} {onSelect} />
		{/each}
		<div class="flex items-center gap-3 text-xs text-muted-foreground">
			<Button size="sm" variant="ghost" disabled={refreshing || generating} onclick={refresh}>
				{refreshing || generating ? 'Updating recommendations…' : 'Refresh recommendations'}
			</Button>
			{#if refreshMessage}<span>{refreshMessage}</span>{/if}
		</div>
	</div>
{/if}

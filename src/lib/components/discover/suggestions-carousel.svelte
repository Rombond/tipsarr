<script lang="ts">
	import { getSuggestions } from '$lib/api/suggestarr';
	import { auth } from '$lib/stores/auth.svelte';
	import { fromSuggestion, type MediaItem } from '$lib/api/media';
	import Carousel from '$lib/components/media/carousel.svelte';

	let { onSelect }: { onSelect?: (item: MediaItem) => void } = $props();

	let items: MediaItem[] = $state([]);
	let loading = $state(true);
	let error: Error | null = $state(null);

	async function load() {
		if (!auth.suggestarrLinked) {
			loading = false;
			items = [];
			return;
		}
		loading = true;
		error = null;
		try {
			const suggestions = await getSuggestions();
			items = (suggestions || []).map(fromSuggestion);
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	load();
</script>

{#if auth.suggestarrLinked}
	<Carousel title="Suggestions" {items} {loading} {error} hasMore={false} {onSelect} onRetry={load} />
{:else}
	<section class="grid gap-2">
		<h2 class="font-semibold text-lg">Suggestions</h2>
		<p class="text-sm text-muted-foreground">
			SuggestArr account not connected.
			{#if auth.suggestarrError}
				<span class="text-destructive">({auth.suggestarrError})</span>
			{/if}
		</p>
	</section>
{/if}

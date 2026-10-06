<script lang="ts">
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let chart = $state<Schemas['Chart'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		unwrap(api.GET('/boxoffice'))
			.then((c) => (chart = c))
			.catch((e) => (error = (e as Error).message))
			.finally(() => (loading = false));
	});

	const money = (n: number) => (n >= 1e6 ? `$${(n / 1e6).toFixed(1)}M` : n > 0 ? `$${Math.round(n / 1000)}K` : '');
	const matched = $derived(chart?.entries.filter((e) => e.item) ?? []);
</script>

{#if loading}
	<Skeleton class="h-64 w-full" />
{:else if error}
	<p class="text-sm text-muted-foreground">Box office unavailable: {error}</p>
{:else if chart && matched.length}
	<section class="grid gap-2">
		<div class="flex items-baseline justify-between">
			<h2 class="font-semibold text-lg">
				Box office <span class="font-normal text-sm text-muted-foreground">· {chart.label} · {chart.region}</span>
			</h2>
			<a class="text-sm underline-offset-2 hover:underline" href="/boxoffice">Full chart & history</a>
		</div>
		<div class="no-scrollbar flex snap-x gap-3 overflow-x-auto pb-2">
			{#each matched as e (e.position)}
				<MediaCard item={e.item!} rank={e.position} note={money(e.weekendGross)} inRadarr={e.inRadarr} />
			{/each}
		</div>
	</section>
{:else if chart}
	<section class="grid gap-1">
		<h2 class="font-semibold text-lg">Box office</h2>
		<p class="text-sm text-muted-foreground">
			No chart stored yet. An admin can fetch it in Settings → Jellyfin sync → boxoffice-refresh.
		</p>
	</section>
{/if}

<style>
	.no-scrollbar {
		scrollbar-width: none;
	}
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
</style>

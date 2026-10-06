<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtMoneyCompact, weekendRange } from '$lib/i18n/format';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let chart = $state<Schemas['Chart'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		unwrap(api.GET('/boxoffice'))
			.then((c) => (chart = c))
			.catch((e) => (error = errorText(e)))
			.finally(() => (loading = false));
	});

	const money = (n: number) => fmtMoneyCompact(n);
	const matched = $derived(chart?.entries.filter((e) => e.item) ?? []);
</script>

{#if loading}
	<Skeleton class="h-64 w-full" />
{:else if error}
	<p class="text-sm text-muted-foreground">{t('boxrow.unavailable', { error })}</p>
{:else if chart && matched.length}
	<section class="grid gap-2">
		<div class="flex items-baseline justify-between gap-3">
			<h2 class="min-w-0 font-semibold text-lg">
				{t('boxrow.title')}
				<span class="block text-xs font-normal text-muted-foreground sm:ml-1 sm:inline sm:text-sm">{weekendRange(chart.week)} · {chart.region}</span>
			</h2>
			<a class="shrink-0 text-sm text-muted-foreground underline-offset-2 hover:text-foreground hover:underline" href="/boxoffice">{t('boxrow.full')}</a>
		</div>
		<div class="no-scrollbar -mx-4 flex snap-x scroll-px-4 gap-3 overflow-x-auto px-4 pb-2 md:-mx-8 md:scroll-px-8 md:px-8">
			{#each matched as e (e.position)}
				<MediaCard item={e.item!} rank={e.position} note={money(e.weekendGross)} inRadarr={e.inRadarr} />
			{/each}
		</div>
	</section>
{:else if chart}
	<section class="grid gap-1">
		<h2 class="font-semibold text-lg">{t('boxrow.title')}</h2>
		<p class="text-sm text-muted-foreground">
			{t('boxrow.empty')}
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

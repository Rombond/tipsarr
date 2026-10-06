<script lang="ts">
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let region = $state('');
	let week = $state('');
	let chart = $state<Schemas['Chart'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			chart = await unwrap(
				api.GET('/boxoffice', { params: { query: { ...(region ? { region } : {}), ...(week ? { week } : {}) } } }),
			);
			region = chart.region;
			week = chart.week;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		region;
		week;
		load();
	});

	const money = (n: number) => (n > 0 ? `$${n.toLocaleString()}` : '-');
	const selectClass = 'h-9 rounded-md border border-input bg-transparent px-3 text-sm';

	function changeRegion(e: Event) {
		week = ''; // each region has its own weeks
		region = (e.currentTarget as HTMLSelectElement).value;
	}
</script>

<svelte:head>
	<title>Box office · Tipsarr</title>
</svelte:head>

<div class="grid max-w-3xl gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="font-bold text-2xl">Box office</h1>
		{#if chart}
			<div class="flex gap-2">
				{#if chart.regions.length > 1}
					<select class={selectClass} value={chart.region} onchange={changeRegion} aria-label="Region">
						{#each chart.regions as r (r)}<option value={r}>{r}</option>{/each}
					</select>
				{/if}
				{#if chart.weeks.length}
					<select class={selectClass} bind:value={week} aria-label="Weekend">
						{#each chart.weeks as w (w.key)}<option value={w.key}>{w.label || w.key}</option>{/each}
					</select>
				{/if}
			</div>
		{/if}
	</div>
	<p class="text-sm text-muted-foreground">
		The same chart for everyone (not personalised). Nothing here is added automatically: open a title to request it.
	</p>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading && !chart}
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-24 w-full" />
	{:else if chart && chart.entries.length === 0}
		<p class="text-sm text-muted-foreground">No chart stored yet. An admin can fetch it in Settings → Jellyfin sync.</p>
	{:else if chart}
		<div class="grid gap-3">
			{#each chart.entries as e (e.position)}
				{@const poster = imageUrl(e.item?.posterPath, 'w154')}
				<div class="flex gap-3 rounded-lg border border-border p-3">
					<span class="w-8 shrink-0 text-center font-black text-2xl text-muted-foreground">{e.position}</span>
					{#if e.item}
						<a href="/media/movie/{e.item.tmdbId}" class="w-14 shrink-0">
							{#if poster}<img src={poster} alt={e.title} class="aspect-[2/3] w-full rounded-md object-cover" loading="lazy" />{/if}
						</a>
					{/if}
					<div class="grid min-w-0 flex-1 content-start gap-1">
						<div class="flex flex-wrap items-center gap-2">
							{#if e.item}
								<a href="/media/movie/{e.item.tmdbId}" class="truncate font-medium hover:underline">{e.title}</a>
							{:else}
								<span class="truncate font-medium">{e.title}</span>
								<Badge variant="outline" class="text-[10px]">no match</Badge>
							{/if}
							{#if e.item?.availability === 'available'}<Badge>Available</Badge>
							{:else if e.item?.requestStatus}<Badge variant="secondary">{e.item.requestStatus === 'pending' ? 'Requested' : 'Approved'}</Badge>
							{:else if e.hasFile}<Badge variant="secondary">In Radarr (file present)</Badge>
							{:else if e.inRadarr}<Badge variant="secondary">In Radarr</Badge>{/if}
						</div>
						<p class="text-xs text-muted-foreground">
							Weekend {money(e.weekendGross)} · Total {money(e.totalGross)} · week {e.weeksInRelease || '-'}
							{#if e.item?.releaseDate}· released {e.item.releaseDate}{/if}
						</p>
					</div>
				</div>
			{/each}
		</div>
		{#if chart.fetchedAt}
			<p class="text-xs text-muted-foreground">Fetched {new Date(chart.fetchedAt * 1000).toLocaleString()} from Box Office Mojo.</p>
		{/if}
	{/if}
</div>

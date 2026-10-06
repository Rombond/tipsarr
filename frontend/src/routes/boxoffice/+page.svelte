<script lang="ts">
	import { api, unwrap, expectOk, imageUrl, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let region = $state('');
	let week = $state('');
	let chart = $state<Schemas['Chart'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// admin: fix a wrong or missing TMDB match
	let fixing = $state<Schemas['ChartEntry'] | null>(null);
	let query = $state('');
	let results = $state<Schemas['Item'][]>([]);
	let searching = $state(false);
	let fixError = $state<string | null>(null);

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

	function startFix(e: Schemas['ChartEntry']) {
		fixing = e;
		query = e.title;
		results = [];
		fixError = null;
		search();
	}

	async function search() {
		if (!query.trim()) return;
		searching = true;
		fixError = null;
		try {
			const r = await unwrap(api.GET('/search', { params: { query: { q: query.trim() } } }));
			results = r.items.filter((i) => i.type === 'movie');
		} catch (e) {
			fixError = (e as Error).message;
		} finally {
			searching = false;
		}
	}

	async function pick(item: Schemas['Item']) {
		if (!fixing) return;
		try {
			await expectOk(api.PUT('/admin/boxoffice/alias', { body: { title: fixing.title, tmdbId: item.tmdbId } }));
			fixing = null;
			await load();
		} catch (e) {
			fixError = (e as Error).message;
		}
	}

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
						{#if auth.isAdmin}
							<div>
								<button type="button" class="cursor-pointer text-xs underline-offset-2 hover:underline" onclick={() => startFix(e)}>
									{e.item ? 'Wrong movie? Fix match' : 'Pick the right movie'}
								</button>
							</div>
						{/if}
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

<Dialog.Root open={fixing !== null} onOpenChange={(o) => !o && (fixing = null)}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>Match "{fixing?.title}" to a TMDB movie</Dialog.Title>
		</Dialog.Header>
		<form
			class="flex gap-2"
			onsubmit={(e) => {
				e.preventDefault();
				search();
			}}
		>
			<Input bind:value={query} placeholder="Search TMDB" />
			<Button type="submit" variant="outline" disabled={searching}>Search</Button>
		</form>
		{#if fixError}<p class="text-sm text-destructive">{fixError}</p>{/if}
		<div class="grid max-h-80 gap-1.5 overflow-y-auto">
			{#each results as r (r.tmdbId)}
				<button
					type="button"
					class="flex cursor-pointer items-center gap-3 rounded-md border border-border p-2 text-left hover:bg-accent"
					onclick={() => pick(r)}
				>
					{#if r.posterPath}<img src={imageUrl(r.posterPath, 'w92')} alt="" class="h-14 w-10 rounded object-cover" />{/if}
					<span class="min-w-0">
						<span class="block truncate text-sm font-medium">{r.title}</span>
						<span class="text-xs text-muted-foreground">{r.releaseDate?.slice(0, 4)} · TMDB {r.tmdbId}</span>
					</span>
				</button>
			{:else}
				{#if !searching}<p class="text-sm text-muted-foreground">No results.</p>{/if}
			{/each}
		</div>
	</Dialog.Content>
</Dialog.Root>

<script lang="ts">
	import { api, unwrap, expectOk, imageUrl, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { toast } from '$lib/toast.svelte';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import CheckIcon from '@lucide/svelte/icons/check';
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

	const compact = (n: number) =>
		n >= 1e6 ? `$${(n / 1e6).toFixed(1)}M` : n >= 1e3 ? `$${Math.round(n / 1e3)}K` : n > 0 ? `$${n}` : '-';
	const top = $derived(Math.max(1, ...(chart?.entries.map((e) => e.weekendGross) ?? [1])));
	const share = (e: Schemas['ChartEntry']) => Math.max(2, Math.round((e.weekendGross / top) * 100));
	const weekIndex = $derived(chart ? chart.weeks.findIndex((w) => w.key === chart!.week) : -1);

	function stepWeek(delta: number) {
		const next = chart?.weeks[weekIndex + delta];
		if (next) week = next.key;
	}

	// what the row can do right now
	function stateOf(e: Schemas['ChartEntry']): 'available' | 'requested' | 'radarr' | 'requestable' | 'none' {
		if (!e.item) return 'none';
		if (e.item.availability === 'available') return 'available';
		if (e.item.requestStatus) return 'requested';
		if (e.inRadarr) return 'radarr';
		return 'requestable';
	}

	let requesting = $state(new Set<number>());
	async function quickRequest(e: Schemas['ChartEntry']) {
		if (!e.item) return;
		const id = e.item.tmdbId;
		requesting = new Set(requesting).add(id);
		try {
			const r = await unwrap(api.POST('/requests', { body: { type: 'movie', tmdbId: id } }));
			toast.success(r.status === 'approved' ? `"${e.title}" approved${r.dryRun ? ' (dry-run: nothing sent)' : ''}` : `Requested "${e.title}"`);
			await load();
		} catch (err) {
			toast.error((err as Error).message);
		} finally {
			const next = new Set(requesting);
			next.delete(id);
			requesting = next;
		}
	}
	const selectClass = 'h-9 rounded-md border border-input bg-transparent px-3 text-sm';

	function changeRegion(e: Event) {
		week = ''; // each region has its own weeks
		region = (e.currentTarget as HTMLSelectElement).value;
	}
</script>

<svelte:head>
	<title>Box office · Tipsarr</title>
</svelte:head>

<div class="grid max-w-4xl gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div>
			<h1 class="font-bold text-2xl">Box office</h1>
			{#if chart?.label}<p class="text-sm text-muted-foreground">Weekend of {chart.label}{chart.regions.length > 1 ? ` · ${chart.region}` : ''}</p>{/if}
		</div>
		{#if chart}
			<div class="flex items-center gap-2">
				{#if chart.regions.length > 1}
					<select class={selectClass} value={chart.region} onchange={changeRegion} aria-label="Region">
						{#each chart.regions as r (r)}<option value={r}>{r}</option>{/each}
					</select>
				{/if}
				{#if chart.weeks.length}
					<Button variant="outline" size="icon" aria-label="Newer weekend" disabled={weekIndex <= 0} onclick={() => stepWeek(-1)}>
						<ChevronLeftIcon class="size-4" />
					</Button>
					<select class={selectClass} bind:value={week} aria-label="Weekend">
						{#each chart.weeks as w (w.key)}<option value={w.key}>{w.label || w.key}</option>{/each}
					</select>
					<Button variant="outline" size="icon" aria-label="Older weekend" disabled={weekIndex >= chart.weeks.length - 1} onclick={() => stepWeek(1)}>
						<ChevronRightIcon class="size-4" />
					</Button>
				{/if}
			</div>
		{/if}
	</div>
	<p class="text-sm text-muted-foreground">The same chart for everyone, not personalised. Nothing is added automatically: request what you want.</p>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading && !chart}
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-24 w-full" />
	{:else if chart && chart.entries.length === 0}
		<div class="rounded-xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
			No chart stored yet.{#if auth.isAdmin} Fetch it in <a class="underline" href="/admin/settings">Settings</a> (job “boxoffice-refresh”).{/if}
		</div>
	{:else if chart}
		<div class="grid gap-2.5" class:opacity-60={loading}>
			{#each chart.entries as e (e.position)}
				{@const poster = imageUrl(e.item?.posterPath, 'w154')}
				{@const state = stateOf(e)}
				<div class="flex gap-3 rounded-xl border border-border bg-card p-3 transition-colors hover:border-foreground/20">
					<span class="w-7 shrink-0 pt-1 text-center font-black text-2xl text-muted-foreground/70">{e.position}</span>
					{#if e.item}
						<a href="/media/movie/{e.item.tmdbId}" class="w-14 shrink-0 sm:w-16">
							{#if poster}<img src={poster} alt="" class="aspect-[2/3] w-full rounded-md object-cover" loading="lazy" />{/if}
						</a>
					{:else}
						<div class="flex aspect-[2/3] w-14 shrink-0 items-center justify-center rounded-md bg-muted text-[10px] text-muted-foreground sm:w-16">?</div>
					{/if}
					<div class="grid min-w-0 flex-1 content-start gap-1.5">
						<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
							{#if e.item}
								<a href="/media/movie/{e.item.tmdbId}" class="truncate font-semibold hover:underline">{e.title}</a>
							{:else}
								<span class="truncate font-semibold">{e.title}</span>
								<Badge variant="outline" class="text-[10px]">no TMDB match</Badge>
							{/if}
							{#if e.item?.releaseDate}<span class="text-xs text-muted-foreground">{e.item.releaseDate.slice(0, 4)}</span>{/if}
						</div>
						<div class="flex items-center gap-2" title="Weekend gross relative to #1">
							<div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
								<div class="h-full rounded-full bg-primary/80" style="width: {share(e)}%"></div>
							</div>
							<span class="w-20 shrink-0 text-right text-xs font-medium tabular-nums">{compact(e.weekendGross)}</span>
						</div>
						<p class="text-xs text-muted-foreground">
							Total {compact(e.totalGross)} · week {e.weeksInRelease || '-'}
						</p>
					</div>
					<div class="flex shrink-0 flex-col items-end justify-between gap-2">
						{#if auth.isAdmin}
							<button
								type="button"
								class="cursor-pointer rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
								title={e.item ? 'Wrong movie? Fix the match' : 'Pick the right movie'}
								aria-label="Fix the TMDB match"
								onclick={() => startFix(e)}
							>
								<PencilIcon class="size-3.5" />
							</button>
						{:else}<span></span>{/if}
						{#if state === 'available'}
							<Badge class="gap-1"><CheckIcon class="size-3" />Available</Badge>
						{:else if state === 'requested'}
							<Badge variant="secondary">{e.item?.requestStatus === 'pending' ? 'Requested' : 'Approved'}</Badge>
						{:else if state === 'radarr'}
							<Badge variant="secondary" title={e.hasFile ? 'Radarr already has the file' : 'Radarr is tracking this movie'}>In Radarr</Badge>
						{:else if state === 'requestable' && e.item}
							<Button size="sm" disabled={requesting.has(e.item.tmdbId)} onclick={() => quickRequest(e)}>
								<PlusIcon class="size-4" />Request
							</Button>
						{/if}
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

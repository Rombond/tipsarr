<script lang="ts">
	import { api, unwrap, expectOk, imageUrl, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { toast } from '$lib/toast.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
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

	function changeRegion(value: string) {
		week = ''; // each region has its own weeks
		region = value;
	}
</script>

<svelte:head>
	<title>Box office · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-5">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="font-bold text-3xl">Box office</h1>
			<p class="text-sm text-muted-foreground">
				{#if chart?.label}Weekend of {chart.label}{chart.regions.length > 1 ? ` · ${chart.region}` : ''} ·{/if}
				the same chart for everyone, nothing is added automatically.
			</p>
		</div>
		{#if chart}
			<div class="flex items-center gap-2">
				{#if chart.regions.length > 1}
					<SimpleSelect
						label="Region"
						value={chart.region}
						options={chart.regions.map((r) => ({ value: r, label: r }))}
						onchange={changeRegion}
						class="min-w-20"
					/>
				{/if}
				{#if chart.weeks.length}
					<!-- the weeks are listed newest first: left = back in time, right = towards today -->
					<Button variant="outline" size="icon" aria-label="Previous (older) weekend" disabled={weekIndex >= chart.weeks.length - 1} onclick={() => stepWeek(1)}>
						<ChevronLeftIcon class="size-4" />
					</Button>
					<SimpleSelect
						label="Weekend"
						value={chart.week}
						options={chart.weeks.map((w) => ({ value: w.key, label: w.label || w.key }))}
						onchange={(v) => (week = v)}
						class="min-w-44"
					/>
					<Button variant="outline" size="icon" aria-label="Next (newer) weekend" disabled={weekIndex <= 0} onclick={() => stepWeek(-1)}>
						<ChevronRightIcon class="size-4" />
					</Button>
				{/if}
			</div>
		{/if}
	</div>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading && !chart}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(11.5rem,1fr))] gap-4">
			{#each { length: 10 } as _, i (i)}<Skeleton class="aspect-[2/3.1] rounded-2xl" />{/each}
		</div>
	{:else if chart && chart.entries.length === 0}
		<div class="rounded-xl border border-dashed border-border p-10 text-center text-sm text-muted-foreground">
			No chart stored yet.{#if auth.isAdmin} Fetch it in <a class="underline" href="/admin/settings">Settings</a> (job “boxoffice-refresh”).{/if}
		</div>
	{:else if chart}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(11.5rem,1fr))] gap-4 transition-opacity" class:opacity-60={loading}>
			{#each chart.entries as e (e.position)}
				{@const poster = imageUrl(e.item?.posterPath, 'w342')}
				{@const state = stateOf(e)}
				<article class="group flex min-w-0 flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg">
					<div class="relative aspect-[2/3] overflow-hidden bg-muted">
						{#if e.item}
							<a href="/media/movie/{e.item.tmdbId}" class="block h-full w-full" aria-label={e.title}>
								{#if poster}<img src={poster} alt="" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]" loading="lazy" />{/if}
							</a>
						{:else}
							<div class="flex h-full w-full items-center justify-center p-3 text-center text-sm text-muted-foreground">{e.title}<br />(no TMDB match)</div>
						{/if}
						<span class="pointer-events-none absolute top-2 left-2 rounded-lg bg-black/70 px-2.5 py-0.5 font-black text-lg text-white backdrop-blur">#{e.position}</span>
						{#if state === 'available'}
							<Badge class="absolute top-2 right-2 gap-1"><CheckIcon class="size-3" />Available</Badge>
						{:else if state === 'requested'}
							<Badge variant="secondary" class="absolute top-2 right-2">{e.item?.requestStatus === 'pending' ? 'Requested' : 'Approved'}</Badge>
						{:else if state === 'radarr'}
							<Badge variant="secondary" class="absolute top-2 right-2" title={e.hasFile ? 'Radarr already has the file' : 'Radarr is tracking this movie'}>In Radarr</Badge>
						{/if}
						{#if auth.isAdmin}
							<button
								type="button"
								class="absolute right-2 bottom-2 flex size-8 cursor-pointer items-center justify-center rounded-full bg-black/70 text-white opacity-0 backdrop-blur transition-opacity group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100"
								title={e.item ? 'Wrong movie? Fix the match' : 'Pick the right movie'}
								aria-label="Fix the TMDB match"
								onclick={() => startFix(e)}
							>
								<PencilIcon class="size-4" />
							</button>
						{/if}
					</div>
					<div class="grid flex-1 content-between gap-3 p-3">
						<div class="grid gap-1.5">
							<h2 class="line-clamp-2 font-semibold leading-snug" title={e.title}>{e.title}</h2>
							<p class="text-xs text-muted-foreground">
								{#if e.item?.releaseDate}{e.item.releaseDate.slice(0, 4)} · {/if}week {e.weeksInRelease || '-'}
							</p>
							<div title="Weekend gross relative to #1">
								<div class="h-1.5 overflow-hidden rounded-full bg-muted">
									<div class="h-full rounded-full bg-primary/80" style="width: {share(e)}%"></div>
								</div>
								<p class="mt-1 flex justify-between text-xs">
									<span class="font-semibold tabular-nums">{compact(e.weekendGross)}</span>
									<span class="text-muted-foreground">total {compact(e.totalGross)}</span>
								</p>
							</div>
						</div>
						{#if state === 'requestable' && e.item}
							<Button size="sm" class="w-full" disabled={requesting.has(e.item.tmdbId)} onclick={() => quickRequest(e)}>
								<PlusIcon class="size-4" />Request
							</Button>
						{:else if e.item}
							<Button size="sm" variant="outline" class="w-full" href="/media/movie/{e.item.tmdbId}">Details</Button>
						{/if}
					</div>
				</article>
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

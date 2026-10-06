<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { dateRange, fmtDateTime, fmtMoneyCompact } from '$lib/i18n/format';
	import { api, unwrap, expectOk, imageUrl, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { toast } from '$lib/toast.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { Badge } from '$lib/components/ui/badge';
	import StatusIcon from '$lib/components/ui/status-icon.svelte';
	import { itemStatus } from '$lib/status';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let region = $state('');
	let weekly = $state(false); // full Monday-Sunday weeks instead of weekends
	let head = $state<Schemas['Chart'] | null>(null); // region/week lists + latest chart
	let charts = $state<Schemas['Chart'][]>([]); // one chart per stored weekend, newest first
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
			const first = await unwrap(api.GET('/boxoffice', { params: { query: { ...(region ? { region } : {}), weekly } } }));
			head = first;
			region = first.region;
			const rest = await Promise.all(
				first.weeks.slice(1).map((w) => unwrap(api.GET('/boxoffice', { params: { query: { region: first.region, week: w.key } } }))),
			);
			charts = first.week ? [first, ...rest] : [];
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		region;
		weekly;
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
			fixError = errorText(e);
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
			fixError = errorText(e);
		}
	}

	const compact = (n: number) => fmtMoneyCompact(n);
	// bars are relative to the #1 of the same weekend
	const share = (c: Schemas['Chart'], e: Schemas['ChartEntry']) =>
		Math.max(2, Math.round((e.weekendGross / Math.max(1, ...c.entries.map((x) => x.weekendGross))) * 100));

	// what the row can do right now: the same states as a request, plus "nothing yet"
	const stateOf = (e: Schemas['ChartEntry']) => (e.item ? itemStatus(e.item, { tracked: e.inRadarr, hasFile: e.hasFile }) : null);

	let requesting = $state(new Set<number>());
	async function quickRequest(e: Schemas['ChartEntry']) {
		if (!e.item) return;
		const id = e.item.tmdbId;
		requesting = new Set(requesting).add(id);
		try {
			const r = await unwrap(api.POST('/requests', { body: { type: 'movie', tmdbId: id } }));
			toast.success(r.status === 'approved' ? t(r.dryRun ? 'media.toast_approved_dry' : 'media.toast_approved', { title: e.title }) : t('media.toast_requested', { title: e.title }));
			await load();
		} catch (err) {
			toast.error(errorText(err));
		} finally {
			const next = new Set(requesting);
			next.delete(id);
			requesting = next;
		}
	}
	const selectClass = 'h-9 rounded-md border border-input bg-transparent px-3 text-sm';

	function changeRegion(value: string) {
		region = value;
	}

	const periods = $derived([
		{ value: 'weekend', label: t('box.period_weekend') },
		{ value: 'week', label: t('box.period_week') },
	]);
</script>

<svelte:head>
	<title>{t('box.page_title')} · Tipsarr</title>
</svelte:head>

{#snippet card(c: Schemas['Chart'], e: Schemas['ChartEntry'])}
	{@const poster = imageUrl(e.item?.posterPath, 'w342')}
	{@const shown = stateOf(e)}
	<article class="group flex min-w-0 flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg">
		<div class="relative aspect-[2/3] overflow-hidden bg-muted">
			{#if e.item}
				<a href="/media/movie/{e.item.tmdbId}" class="block h-full w-full" aria-label={e.title}>
					{#if poster}<img src={poster} alt="" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]" loading="lazy" />{/if}
				</a>
			{:else}
				<div class="flex h-full w-full items-center justify-center p-3 text-center text-sm text-muted-foreground">{t('box.no_match', { title: e.title })}</div>
			{/if}
			<span class="pointer-events-none absolute top-2 left-2 rounded-lg bg-black/70 px-2.5 py-0.5 font-black text-lg text-white backdrop-blur">#{e.position}</span>
			{#if shown}<StatusIcon status={shown} class="absolute top-2 right-2" />{/if}
			{#if auth.isAdmin}
				<button
					type="button"
					class="absolute right-2 bottom-2 flex size-8 cursor-pointer items-center justify-center rounded-full bg-black/70 text-white opacity-0 backdrop-blur transition-opacity group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100"
					title={e.item ? t('box.fix_wrong') : t('box.fix_pick')}
					aria-label={t('box.fix_aria')}
					onclick={() => startFix(e)}
				>
					<PencilIcon class="size-4" />
				</button>
			{/if}
		</div>
		<div class="grid flex-1 content-between gap-3 p-3">
			<div class="grid gap-1.5">
				<h3 class="line-clamp-2 font-semibold leading-snug" title={e.title}>{e.title}</h3>
				<p class="text-xs text-muted-foreground">
					{#if e.item?.releaseDate}{e.item.releaseDate.slice(0, 4)}&nbsp;·&nbsp;{/if}{t('box.week_n', { n: e.weeksInRelease || '-' })}
				</p>
				<div title={c.weekly ? t('box.gross_hint_week') : t('box.gross_hint')}>
					<p class="flex items-baseline justify-between gap-2">
						<span class="font-semibold text-sm tabular-nums">{compact(e.weekendGross)}</span>
						<span class="text-xs text-muted-foreground">{c.weekly ? t('box.this_week') : t('box.this_weekend')}</span>
					</p>
					<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
						<div class="h-full rounded-full bg-primary/80" style="width: {share(c, e)}%"></div>
					</div>
					<p class="mt-1 text-xs text-muted-foreground">{t('box.total', { amount: compact(e.totalGross) })}</p>
				</div>
			</div>
			{#if shown === null && e.item}
				<Button size="sm" class="w-full" disabled={requesting.has(e.item.tmdbId)} onclick={() => quickRequest(e)}>
					<PlusIcon class="size-4" />{t('media.request')}
				</Button>
			{:else if e.item}
				<Button size="sm" variant="outline" class="w-full" href="/media/movie/{e.item.tmdbId}">{t('common.details')}</Button>
			{/if}
		</div>
	</article>
{/snippet}

<div class="grid grid-cols-[minmax(0,1fr)] gap-8">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="font-bold text-3xl">{t('box.title')}</h1>
			<p class="text-sm text-muted-foreground">{t('box.subtitle')}</p>
			<p class="text-xs text-muted-foreground">{t('box.bar_hint')}</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if head && head.regions.length > 1}
				<SimpleSelect
					label={t('box.region')}
					value={head.region}
					options={head.regions.map((r) => ({ value: r, label: r }))}
					onchange={changeRegion}
					class="min-w-20"
				/>
			{/if}
			{#if head?.hasWeekly || weekly}
				<SimpleSelect label={t('box.period')} value={weekly ? 'week' : 'weekend'} options={periods} onchange={(v) => (weekly = v === 'week')} class="min-w-32" />
			{/if}
			{#if charts.length > 1}
				<!-- jump chips: the weekends sit under each other, these scroll to them -->
				<nav class="flex flex-wrap gap-1.5" aria-label={t('box.jump')}>
					{#each charts as c (c.week)}
						<a href="#w-{c.week}" class="rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground hover:bg-accent hover:text-foreground">{dateRange(c.start, c.end)}</a>
					{/each}
				</nav>
			{/if}
		</div>
	</div>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if loading && charts.length === 0}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(11.5rem,1fr))] gap-4">
			{#each { length: 10 } as _, i (i)}<Skeleton class="aspect-[2/3.1] rounded-2xl" />{/each}
		</div>
	{:else if charts.length === 0}
		<div class="rounded-xl border border-dashed border-border p-10 text-center text-sm text-muted-foreground">
			{t('box.empty')}{#if auth.isAdmin} <a class="underline" href="/admin/settings">{t('box.empty_admin')}</a>{/if}
		</div>
	{:else}
		{#each charts as c, i (c.week)}
			<section id="w-{c.week}" class="grid scroll-mt-24 gap-4 transition-opacity" class:opacity-60={loading}>
				<div class="flex items-baseline gap-3 border-b border-border pb-2">
					<h2 class="font-semibold text-xl">{t(c.weekly ? 'box.week_of' : 'box.weekend_of', { range: dateRange(c.start, c.end) })}</h2>
					{#if i === 0}<Badge variant="secondary">{t('box.latest')}</Badge>{/if}
					<span class="ml-auto text-xs text-muted-foreground">{c.region}</span>
				</div>
				{#if c.entries.length === 0}
					<p class="text-sm text-muted-foreground">{t('box.no_entries')}</p>
				{:else}
					<div class="grid grid-cols-[repeat(auto-fill,minmax(11.5rem,1fr))] gap-4">
						{#each c.entries as e (e.position)}{@render card(c, e)}{/each}
					</div>
				{/if}
			</section>
		{/each}
		{#if head?.fetchedAt}
			<p class="text-xs text-muted-foreground">{t('box.fetched', { when: fmtDateTime(head.fetchedAt) })}</p>
		{/if}
	{/if}
</div>

<Dialog.Root open={fixing !== null} onOpenChange={(o) => !o && (fixing = null)}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{t('box.fix_title', { title: fixing?.title ?? '' })}</Dialog.Title>
		</Dialog.Header>
		<form
			class="flex gap-2"
			onsubmit={(e) => {
				e.preventDefault();
				search();
			}}
		>
			<Input bind:value={query} placeholder={t('box.fix_search')} />
			<Button type="submit" variant="outline" disabled={searching}>{t('box.fix_go')}</Button>
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
				{#if !searching}<p class="text-sm text-muted-foreground">{t('common.no_results')}</p>{/if}
			{/each}
		</div>
	</Dialog.Content>
</Dialog.Root>

<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { i18n } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import BarChart from '$lib/components/stats/bar-chart.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import InfoIcon from '@lucide/svelte/icons/info';
	import ExternalIcon from '@lucide/svelte/icons/external-link';

	type Period = '30d' | '12m' | 'all';
	let user = $state(''); // '' = me, 'all' = everyone, else a user id (admins)
	let period = $state<Period>('all');
	let report = $state<Schemas['StatsReport'] | null>(null);
	let users = $state<Schemas['User'][]>([]);
	let jellyfinLink = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			report = await unwrap(api.GET('/stats', { params: { query: { user: user || undefined, period } } }));
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		if (auth.isAdmin) {
			unwrap(api.GET('/admin/users')).then((u) => (users = u)).catch(() => {});
			unwrap(api.GET('/admin/settings')).then((s) => (jellyfinLink = (s.jellyfinPublicUrl || s.jellyfinUrl).replace(/\/$/, ''))).catch(() => {});
		}
	});

	// reload when the person or the period changes
	$effect(() => {
		user;
		period;
		load();
	});

	const num = (n: number, digits = 0) => new Intl.NumberFormat(i18n.tag, { maximumFractionDigits: digits }).format(n);
	const hours = (h: number) => `${num(h, h >= 100 ? 0 : 1)} ${t('stats.unit_h')}`;
	const plugin = $derived(report?.source === 'plugin');
	const month = (ym: string) => new Intl.DateTimeFormat(i18n.tag, { month: 'short' }).format(new Date(`${ym}-15T12:00:00`));
	const weekday = (i: number) => new Intl.DateTimeFormat(i18n.tag, { weekday: 'short' }).format(new Date(2024, 0, 1 + i)); // 2024-01-01 is a Monday

	const userOptions = $derived([
		{ value: '', label: t('stats.me') },
		{ value: 'all', label: t('stats.everyone') },
		...users.filter((u) => u.id !== auth.user?.id).map((u) => ({ value: u.id, label: u.name })),
	]);
	const periodOptions = $derived([
		{ value: 'all', label: t('stats.period_all') },
		{ value: '12m', label: t('stats.period_12m') },
		{ value: '30d', label: t('stats.period_30d') },
	]);

	const genreData = $derived((report?.genres ?? []).map((g) => ({ label: g.name, value: g.hours || g.titles, text: g.hours ? hours(g.hours) : t('stats.n_titles', { count: g.titles }) })));
	const decadeData = $derived((report?.decades ?? []).map((d) => ({ label: d.name, value: d.titles, text: t('stats.n_titles', { count: d.titles }) })));
	const monthData = $derived((report?.months ?? []).map((m) => ({ label: month(m.month), value: m.hours, text: hours(m.hours) })));
	const weekdayData = $derived((report?.weekdays ?? []).map((h, i) => ({ label: weekday(i), value: h, text: hours(h) })));
	const hourData = $derived((report?.hoursOfDay ?? []).map((h, i) => ({ label: String(i), value: h, text: `${i}:00 · ${hours(h)}` })));
	const empty = $derived(!!report && report.totals.titles === 0);
</script>

<svelte:head>
	<title>{t('stats.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-5">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div class="grid gap-1">
			<h1 class="text-3xl font-bold">{t('stats.title')}</h1>
			<p class="text-sm text-muted-foreground">
				{#if report}{plugin ? t('stats.source_plugin') : t('stats.source_estimate')}{:else}&nbsp;{/if}
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if auth.isAdmin && users.length}
				<SimpleSelect label={t('stats.user')} value={user} options={userOptions} onchange={(v) => (user = v)} class="w-44" />
			{/if}
			{#if plugin}
				<SimpleSelect label={t('stats.period')} value={period} options={periodOptions} onchange={(v) => (period = v as Period)} class="w-44" />
			{/if}
		</div>
	</div>

	{#if error}
		<p class="text-sm text-destructive">{t('common.error_prefix', { message: error })} <button class="ml-2 underline" onclick={load}>{t('common.retry')}</button></p>
	{:else if loading && !report}
		<div class="grid grid-cols-2 gap-3 lg:grid-cols-4">{#each { length: 4 } as _, i (i)}<Skeleton class="h-24 rounded-xl" />{/each}</div>
	{:else if report}
		{#if report.plugin.hint}
			<Card class="border-primary/40">
				<CardHeader>
					<CardTitle class="flex items-center gap-2"><InfoIcon class="size-4" />{t('stats.plugin_title')}</CardTitle>
					<CardDescription>{t('stats.plugin_text')}</CardDescription>
				</CardHeader>
				<CardContent class="grid gap-3 text-sm">
					<ol class="list-decimal space-y-1 pl-5 text-muted-foreground">
						<li>{t('stats.plugin_step1')}</li>
						<li>{t('stats.plugin_step2')}</li>
						<li>{t('stats.plugin_step3')}</li>
					</ol>
					<div class="flex flex-wrap gap-2">
						{#if jellyfinLink}<Button variant="outline" size="sm" href="{jellyfinLink}/web/#/dashboard/plugins" target="_blank" rel="noreferrer"><ExternalIcon /> {t('stats.plugin_open')}</Button>{/if}
						<Button variant="ghost" size="sm" href="https://github.com/jellyfin/jellyfin-plugin-playbackreporting" target="_blank" rel="noreferrer">{t('stats.plugin_about')}</Button>
					</div>
				</CardContent>
			</Card>
		{/if}

		{#if empty}
			<p class="text-sm text-muted-foreground">{t('stats.empty')}</p>
		{:else}
			<div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
				<div class="rounded-xl border border-border p-4">
					<p class="text-xs text-muted-foreground">{plugin ? t('stats.tile_hours') : t('stats.tile_hours_est')}</p>
					<p class="mt-1 text-3xl font-bold tabular-nums">{num(report.totals.hours, report.totals.hours >= 100 ? 0 : 1)}</p>
					<p class="text-xs text-muted-foreground">{t('stats.unit_hours_long')}</p>
				</div>
				<div class="rounded-xl border border-border p-4">
					<p class="text-xs text-muted-foreground">{t('stats.tile_titles')}</p>
					<p class="mt-1 text-3xl font-bold tabular-nums">{num(report.totals.titles)}</p>
					<p class="text-xs text-muted-foreground">{t('stats.tile_titles_sub', { movies: report.totals.movies, shows: report.totals.shows })}</p>
				</div>
				<div class="rounded-xl border border-border p-4">
					<p class="text-xs text-muted-foreground">{t('stats.tile_plays')}</p>
					<p class="mt-1 text-3xl font-bold tabular-nums">{num(report.totals.plays)}</p>
					<p class="text-xs text-muted-foreground">{report.genres[0] ? t('stats.tile_plays_sub', { genre: report.genres[0].name }) : ' '}</p>
				</div>
				<div class="rounded-xl border border-border p-4">
					<p class="text-xs text-muted-foreground">{t('stats.tile_requests')}</p>
					<p class="mt-1 text-3xl font-bold tabular-nums">{num(report.requests.made)}</p>
					<p class="text-xs text-muted-foreground">{t('stats.tile_requests_sub', { available: report.requests.available, declined: report.requests.declined })}</p>
				</div>
			</div>

			<div class="grid gap-4 lg:grid-cols-2">
				{#if genreData.length}
					<Card>
						<CardHeader><CardTitle>{t('stats.genres')}</CardTitle></CardHeader>
						<CardContent><BarChart data={genreData} caption={t('stats.genres')} /></CardContent>
					</Card>
				{/if}
				{#if decadeData.length}
					<Card>
						<CardHeader><CardTitle>{t('stats.decades')}</CardTitle></CardHeader>
						<CardContent><BarChart data={decadeData} direction="vertical" caption={t('stats.decades')} /></CardContent>
					</Card>
				{/if}
				{#if plugin}
					<Card class="lg:col-span-2">
						<CardHeader><CardTitle>{t('stats.months')}</CardTitle></CardHeader>
						<CardContent><BarChart data={monthData} direction="vertical" caption={t('stats.months')} /></CardContent>
					</Card>
					<Card>
						<CardHeader><CardTitle>{t('stats.weekdays')}</CardTitle></CardHeader>
						<CardContent><BarChart data={weekdayData} direction="vertical" caption={t('stats.weekdays')} /></CardContent>
					</Card>
					<Card>
						<CardHeader><CardTitle>{t('stats.hours_of_day')}</CardTitle></CardHeader>
						<CardContent><BarChart data={hourData} direction="vertical" caption={t('stats.hours_of_day')} labelEvery={3} /></CardContent>
					</Card>
				{/if}
			</div>

			{#if report.top.length}
				<Card>
					<CardHeader><CardTitle>{t('stats.top')}</CardTitle></CardHeader>
					<CardContent>
						<ol class="grid grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] gap-4">
							{#each report.top as item, i (item.type + item.tmdbId)}
								<li class="min-w-0">
									<a href="/media/{item.type}/{item.tmdbId}" class="group block">
										<div class="relative aspect-[2/3] overflow-hidden rounded-lg bg-muted ring-1 ring-border/50 transition group-hover:ring-border">
											{#if item.posterUrl}<img src={item.posterUrl} alt="" loading="lazy" class="h-full w-full object-cover" />{:else}<div class="flex h-full items-center justify-center p-2 text-center text-xs text-muted-foreground">{item.title}</div>{/if}
											<span class="absolute top-1.5 left-1.5 rounded-full bg-black/70 px-1.5 text-[11px] font-medium text-white">{i + 1}</span>
										</div>
										<p class="mt-1.5 truncate text-sm font-medium" title={item.title}>{item.title}</p>
										<p class="truncate text-xs text-muted-foreground">{t('stats.n_plays', { count: item.plays })}{item.hours ? ` · ${hours(item.hours)}` : ''}</p>
									</a>
								</li>
							{/each}
						</ol>
					</CardContent>
				</Card>
			{/if}
		{/if}
	{/if}
</div>

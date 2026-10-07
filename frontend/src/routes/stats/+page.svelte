<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { i18n } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, imageUrl, type Schemas } from '$lib/api/client';
	import { fmtDate } from '$lib/i18n/format';
	import { auth } from '$lib/stores/auth.svelte';
	import BarChart from '$lib/components/stats/bar-chart.svelte';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import Scroller from '$lib/components/ui/scroller.svelte';
	import { asMediaItem } from '$lib/library-item';
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
	let unwatched = $state<Schemas['View'][]>([]);
	let unwatchedTotal = $state(0);
	let neverWatched = $state<Schemas['LibraryItem'][]>([]);
	let neverTotal = $state(0);
	let users = $state<Schemas['User'][]>([]);
	let jellyfinLink = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			report = await unwrap(api.GET('/stats', { params: { query: { user: user || undefined, period } } }));
			// what they asked for, is available, and never watched: an admin looking at everyone sees everyone's
			const who = user === 'all' ? undefined : user || auth.user?.id;
			const mine = await unwrap(api.GET('/requests', { params: { query: { filter: 'unwatched', user: who, take: 40 } } })).catch(() => null);
			unwatched = mine?.items ?? [];
			unwatchedTotal = mine?.total ?? 0;
			// library clean-up (admins): titles nobody ever watched, oldest first
			if (auth.isAdmin) {
				const lib = await unwrap(api.GET('/library', { params: { query: { neverWatched: true, sort: 'added', dir: 'asc', pageSize: 30 } } })).catch(() => null);
				neverWatched = lib?.items ?? [];
				neverTotal = lib?.total ?? 0;
			}
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

			{#snippet carousel(title: string, list: Schemas['StatsTop'][])}
				{#if list.length}
					<Scroller {title}>
						{#each list as item, i (item.type + item.tmdbId)}
							<MediaCard
								item={asMediaItem(item)}
								posterUrl={item.posterUrl}
								hideStatus
								rank={i + 1}
								note={`${hours(item.hours)} · ${item.type === 'tv' ? t('stats.n_episodes', { count: item.plays }) : t('stats.n_plays', { count: item.plays })}`}
							/>
						{/each}
					</Scroller>
				{/if}
			{/snippet}
			{@render carousel(t('stats.top'), report.top)}
			{@render carousel(t('stats.top_movies'), report.topMovies)}
			{@render carousel(t('stats.top_shows'), report.topShows)}

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

		{/if}

		<Card>
			<CardHeader>
				<CardTitle>{t('stats.unwatched_title')}</CardTitle>
				<CardDescription>{user === '' || !auth.isAdmin ? t('stats.unwatched_mine') : t('stats.unwatched_others')}</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-2">
				{#each unwatched as r (r.id)}
					<a href="/media/{r.type}/{r.tmdbId}" class="flex items-center gap-3 rounded-lg border border-border p-2 hover:bg-accent">
						<div class="h-14 w-10 shrink-0 overflow-hidden rounded bg-muted">
							{#if r.posterPath}<img src={imageUrl(r.posterPath, 'w92')} alt="" loading="lazy" class="h-full w-full object-cover" />{/if}
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium">{r.title}</p>
							<p class="truncate text-xs text-muted-foreground">
								{r.type === 'tv' ? t('type.tv_short') : t('type.movie')}
								{#if user === 'all' || (auth.isAdmin && user !== '')} · {r.requestedBy?.name ?? ''}{/if}
								· {t('stats.unwatched_since', { date: fmtDate(r.updatedAt) })}
							</p>
						</div>
					</a>
				{:else}
					<p class="text-sm text-muted-foreground">{t('stats.unwatched_none')}</p>
				{/each}
				{#if unwatchedTotal > unwatched.length}
					<p class="text-xs text-muted-foreground">{t('stats.unwatched_more', { count: unwatchedTotal - unwatched.length })}</p>
				{/if}
			</CardContent>
		</Card>

		{#if auth.isAdmin}
			{#if neverWatched.length}
				<Scroller title={t('stats.never_title')}>
					{#each neverWatched as item (item.type + item.tmdbId)}
						<MediaCard
							item={asMediaItem(item)}
							posterUrl={item.posterUrl}
							hideStatus
							note={item.addedAt ? t('library.added_on', { date: fmtDate(item.addedAt) }) : undefined}
						/>
					{/each}
				</Scroller>
				<p class="-mt-4 text-xs text-muted-foreground">
					{t('stats.never_hint', { count: neverTotal })}
					<a href="/library?never=1" class="underline-offset-4 hover:underline">{t('stats.never_open')}</a>
				</p>
			{:else}
				<Card>
					<CardHeader>
						<CardTitle>{t('stats.never_title')}</CardTitle>
						<CardDescription>{t('stats.never_none')}</CardDescription>
					</CardHeader>
				</Card>
			{/if}
		{/if}
	{/if}
</div>

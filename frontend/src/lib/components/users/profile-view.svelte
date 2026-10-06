<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import UserAvatar from './user-avatar.svelte';
	import ProfileSettings from './profile-settings.svelte';
	import RequestRow from '$lib/components/requests/request-row.svelte';
	import Carousel from '$lib/components/media/carousel.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import FilmIcon from '@lucide/svelte/icons/film';
	import TvIcon from '@lucide/svelte/icons/tv';
	import ListIcon from '@lucide/svelte/icons/list';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import EyeIcon from '@lucide/svelte/icons/eye';

	let { userId, self = false }: { userId: string; self?: boolean } = $props();

	let profile = $state<Schemas['ProfileView'] | null>(null);
	let recent = $state<Schemas['View'][]>([]);
	let watchlist = $state<Schemas['Item'][]>([]);
	let error = $state<string | null>(null);
	let loading = $state(true);
	let tab = $state<'overview' | 'settings'>('overview');

	$effect(() => {
		const id = userId;
		loading = true;
		error = null;
		Promise.all([
			unwrap(api.GET('/users/{id}', { params: { path: { id } } })),
			unwrap(api.GET('/requests', { params: { query: { take: 6, ...(self ? { filter: 'mine' as const } : { user: id }) } } })),
			self ? unwrap(api.GET('/watchlist')) : Promise.resolve([] as Schemas['Item'][]),
		])
			.then(([p, r, w]) => {
				profile = p;
				recent = r.items;
				watchlist = w;
			})
			.catch((e) => (error = errorText(e)))
			.finally(() => (loading = false));
	});

	type Tile = { label: string; value: number; dot?: string; icon?: typeof FilmIcon };
	const tiles = $derived<Tile[]>(
		profile
			? [
					{ label: t('profile.stat_requests'), value: profile.stats.requests, icon: ListIcon },
					{ label: t('type.movies'), value: profile.stats.movies, icon: FilmIcon },
					{ label: t('type.shows'), value: profile.stats.shows, icon: TvIcon },
					{ label: t('requests.tab.pending'), value: profile.stats.pending, dot: 'bg-amber-500' },
					{ label: t('requests.tab.approved'), value: profile.stats.approved, dot: 'bg-sky-500' },
					{ label: t('requests.tab.available'), value: profile.stats.available, dot: 'bg-emerald-500' },
					{ label: t('requests.tab.declined'), value: profile.stats.declined, dot: 'bg-rose-500' },
					{ label: t('requests.tab.failed'), value: profile.stats.failed, dot: 'bg-orange-600' },
					{ label: t('profile.stat_watchlist'), value: profile.stats.watchlist, icon: BookmarkIcon },
					{ label: t('profile.stat_watched'), value: profile.stats.watched, icon: EyeIcon },
				]
			: [],
	);
</script>

{#if loading}
	<div class="grid gap-4">
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-40 w-full" />
	</div>
{:else if error}
	<p class="text-sm text-destructive">{error}</p>
{:else if profile}
	<div class="grid grid-cols-[minmax(0,1fr)] gap-6">
		<header class="flex flex-wrap items-center gap-4 rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-card to-card p-5">
			<UserAvatar id={profile.id} name={profile.name} class="size-20 text-2xl" />
			<div class="grid gap-1">
				<h1 class="font-bold text-3xl">{profile.name}</h1>
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
					<Badge variant={profile.role === 'admin' ? 'default' : 'secondary'}>{profile.role === 'admin' ? t('users.role_admin') : t('users.role_user')}</Badge>
					{#if profile.createdAt}<span>{t('profile.member_since', { date: fmtDate(profile.createdAt) })}</span>{/if}
					{#if profile.lastLoginAt}<span>{t('profile.last_seen', { date: fmtDate(profile.lastLoginAt) })}</span>{/if}
					{#if profile.region}<span>{profile.region}</span>{/if}
				</div>
			</div>
			{#if !self && auth.isAdmin}
				<a class="ml-auto text-sm underline" href="/admin/users">{t('profile.manage_user')}</a>
			{/if}
		</header>

		{#if self}
			<div class="flex gap-1" role="tablist" aria-label={t('profile.title')}>
				{#each [['overview', t('profile.tab_overview')], ['settings', t('profile.tab_settings')]] as [id, name] (id)}
					<button
						type="button"
						role="tab"
						aria-selected={tab === id}
						class="cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors {tab === id ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
						onclick={() => (tab = id as 'overview' | 'settings')}
					>{name}</button>
				{/each}
			</div>
		{/if}

		{#if tab === 'settings' && self}
			<ProfileSettings />
		{:else}
			<section class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5" aria-label={t('profile.stats')}>
				{#each tiles as tile (tile.label)}
					<div class="rounded-xl border border-border bg-card p-4">
						<p class="flex items-center gap-1.5 text-xs text-muted-foreground">
							{#if tile.dot}<span class="size-2 rounded-full {tile.dot}"></span>{:else if tile.icon}<tile.icon class="size-3.5" />{/if}
							{tile.label}
						</p>
						<p class="mt-1 font-bold text-2xl tabular-nums">{tile.value}</p>
					</div>
				{/each}
			</section>

			<section class="grid gap-3">
				<div class="flex items-baseline justify-between">
					<h2 class="font-semibold text-lg">{t('profile.recent_requests')}</h2>
					{#if self}<a href="/requests" class="text-sm text-muted-foreground underline-offset-2 hover:text-foreground hover:underline">{t('profile.all_requests')}</a>{/if}
				</div>
				{#if recent.length}
					<div class="grid gap-3 xl:grid-cols-2">
						{#each recent as r (r.id)}
							<RequestRow request={r} isAdmin={false} />
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">{t('profile.no_requests')}</p>
				{/if}
			</section>

			{#if self && watchlist.length}
				<Carousel title={t('watchlist.title')} items={watchlist} />
			{/if}
		{/if}
	</div>
{/if}

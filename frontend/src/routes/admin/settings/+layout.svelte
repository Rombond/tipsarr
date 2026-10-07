<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t, type Key } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { onEvent } from '$lib/events.svelte';

	let { children } = $props();

	type Dot = 'warn' | 'error' | null;
	const tabs = $derived<{ id: string; key: Key; dot: Dot }[]>([
		{ id: 'connections', key: 'settings.tab_connections', dot: admin.settings && !(admin.settings.tmdbConfigured && admin.settings.jellyfinApiKeyConfigured) ? 'warn' : null },
		{ id: 'sign-in', key: 'settings.tab_signin', dot: null },
		{ id: 'requests', key: 'settings.tab_requests', dot: null },
		{ id: 'general', key: 'settings.tab_general', dot: null },
		{ id: 'jobs', key: 'settings.tab_jobs', dot: admin.sync?.jobs.some((j) => j.status === 'error') ? 'error' : null },
	]);

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		admin.loadSettings();
		admin.refreshSync();
		// the server pushes job changes; polling is the safety net while something runs
		const off = onEvent('sync.status', () => admin.refreshSync());
		const timer = setInterval(() => {
			if (admin.anyRunning || Object.keys(admin.started).length) admin.refreshSync();
		}, 1500);
		return () => {
			clearInterval(timer);
			off();
		};
	});
</script>

<svelte:head>
	<title>{t('settings.title')} · Tipsarr</title>
</svelte:head>

<div class="grid gap-5">
	<h1 class="text-3xl font-bold">{t('settings.title')}</h1>

	<div class="grid items-start gap-5 md:grid-cols-[11rem_minmax(0,1fr)]">
		<nav class="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1 md:sticky md:top-4 md:mx-0 md:flex-col md:overflow-visible md:px-0 md:pb-0" aria-label={t('settings.tabs_aria')}>
			{#each tabs as tab (tab.id)}
				{@const active = page.url.pathname.startsWith(`/admin/settings/${tab.id}`)}
				<a
					href="/admin/settings/{tab.id}"
					aria-current={active ? 'page' : undefined}
					class="flex shrink-0 items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm whitespace-nowrap transition-colors {active ? 'bg-accent font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'}"
				>
					{t(tab.key)}
					{#if tab.dot}
						<span class="size-2 rounded-full {tab.dot === 'error' ? 'bg-destructive' : 'bg-amber-500'}" aria-hidden="true"></span>
						<span class="sr-only">{tab.dot === 'error' ? t('settings.tab_error') : t('settings.tab_todo')}</span>
					{/if}
				</a>
			{/each}
		</nav>

		<div class="grid min-w-0 max-w-3xl gap-4">
			{#if admin.error}<p class="text-sm text-destructive">{admin.error}</p>{/if}
			{@render children()}
		</div>
	</div>
</div>

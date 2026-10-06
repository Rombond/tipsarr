<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';

	let settings = $state<Schemas['SettingsBody'] | null>(null);
	let sync = $state<Schemas['SyncStatusBody'] | null>(null);
	let tmdbKey = $state('');
	let jellyfinKey = $state('');
	let regions = $state('');
	let message = $state<string | null>(null);
	let error = $state<string | null>(null);
	let saving = $state(false);

	const anyRunning = $derived(sync?.jobs.some((j) => j.running) ?? false);

	async function refreshSync() {
		try {
			sync = await unwrap(api.GET('/admin/sync'));
		} catch (e) {
			error = (e as Error).message;
		}
	}

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		unwrap(api.GET('/admin/settings'))
			.then((s) => {
				settings = s;
				regions = s.boxofficeRegions;
			})
			.catch((e) => (error = (e as Error).message));
		refreshSync();
		const timer = setInterval(() => {
			if (anyRunning) refreshSync();
		}, 2000);
		return () => clearInterval(timer);
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = message = null;
		try {
			settings = await unwrap(
				api.PUT('/admin/settings', {
					body: {
						...(tmdbKey.trim() ? { tmdbApiKey: tmdbKey.trim() } : {}),
						...(jellyfinKey.trim() ? { jellyfinApiKey: jellyfinKey.trim() } : {}),
					},
				}),
			);
			tmdbKey = jellyfinKey = '';
			message = 'Saved.';
			await refreshSync();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			saving = false;
		}
	}

	async function saveRegions(e: SubmitEvent) {
		e.preventDefault();
		error = message = null;
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { boxofficeRegions: regions } }));
			regions = settings.boxofficeRegions;
			message = 'Box office regions saved. Run boxoffice-refresh below to fetch them.';
		} catch (err) {
			error = (err as Error).message;
		}
	}

	type JobName = 'library-sync' | 'history-sync' | 'boxoffice-refresh';

	async function runJob(job: JobName) {
		error = message = null;
		try {
			await unwrap(api.POST('/admin/sync/{job}', { params: { path: { job } } }));
			await refreshSync();
		} catch (err) {
			error = (err as Error).message;
		}
	}

	const when = (unix: number) => (unix ? new Date(unix * 1000).toLocaleString() : 'never');
	const webhookUrl = $derived(settings ? `${location.origin}${settings.webhookPath}` : '');
</script>

<svelte:head>
	<title>Settings · Tipsarr</title>
</svelte:head>

<div class="grid max-w-2xl gap-4">
	<h1 class="font-bold text-2xl">Settings</h1>

	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}

	{#if settings}
		<Card>
			<CardHeader>
				<CardTitle>Status</CardTitle>
			</CardHeader>
			<CardContent class="grid gap-1.5 text-sm">
				<div class="flex justify-between border-b border-border py-1.5">
					<span class="text-muted-foreground">Jellyfin</span>
					<span class="font-mono">{settings.jellyfinUrl}</span>
				</div>
				<div class="flex items-center justify-between py-1.5">
					<span class="text-muted-foreground">Dry-run</span>
					<Badge variant={settings.dryRun ? 'default' : 'destructive'}>{settings.dryRun ? 'ON (safe)' : 'OFF'}</Badge>
				</div>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>API keys</CardTitle>
				<CardDescription>
					TMDB: {settings.tmdbConfigured ? 'saved' : 'missing (discover and search need it)'}. Jellyfin API key:
					{settings.jellyfinApiKeyConfigured ? 'saved' : 'missing (library sync needs it; create one in Jellyfin: Dashboard → API Keys)'}.
					Entering a value replaces the saved one; keys are never shown again.
				</CardDescription>
			</CardHeader>
			<CardContent>
				<form class="grid gap-3" onsubmit={save}>
					<Input placeholder="TMDB API key or read-access token" bind:value={tmdbKey} autocomplete="off" />
					<Input placeholder="Jellyfin API key" bind:value={jellyfinKey} autocomplete="off" />
					{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}
					<Button type="submit" disabled={saving || (!tmdbKey.trim() && !jellyfinKey.trim())}>
						{saving ? 'Saving…' : 'Save'}
					</Button>
				</form>
			</CardContent>
		</Card>
	{/if}

	{#if settings}
		<Card>
			<CardHeader>
				<CardTitle>Box office regions</CardTitle>
				<CardDescription>
					Comma-separated codes, e.g. <code>US,GB,FR</code> (US = United States &amp; Canada). Charts come from Box Office Mojo's public
					pages and are fetched every 12 hours.
				</CardDescription>
			</CardHeader>
			<CardContent>
				<form class="flex gap-2" onsubmit={saveRegions}>
					<Input bind:value={regions} placeholder="US" autocomplete="off" />
					<Button type="submit" variant="outline">Save</Button>
				</form>
			</CardContent>
		</Card>
	{/if}

	{#if sync}
		<Card>
			<CardHeader>
				<CardTitle>Background jobs</CardTitle>
				<CardDescription>
					Jellyfin sync is read-only. {sync.movies} movies and {sync.shows} shows known in the library.
					{#if !sync.canSync}Save a Jellyfin API key above to enable syncing.{/if}
				</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-3 text-sm">
				{#each sync.jobs as job (job.name)}
					<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3">
						<div class="grid gap-0.5">
							<span class="font-medium">{job.name}</span>
							<span class="text-xs text-muted-foreground">
								{job.running ? 'running…' : `${job.status} · ${when(job.lastFinishedAt)}`}
								{#if job.message}· {job.message}{/if}
							</span>
							<span class="text-xs text-muted-foreground">runs every {Math.round(job.everySeconds / 3600)} h</span>
						</div>
						<Button
							variant="outline"
							size="sm"
							disabled={(job.name !== 'boxoffice-refresh' && !sync.canSync) || job.running}
							onclick={() => runJob(job.name as JobName)}
						>
							Sync now
						</Button>
					</div>
				{/each}
				{#if anyRunning}<p class="text-xs text-muted-foreground">Refreshing…</p>{/if}
			</CardContent>
		</Card>

		{#if settings}
			<Card>
				<CardHeader>
					<CardTitle>Jellyfin webhook (optional)</CardTitle>
					<CardDescription>
						Makes watch history and "available" badges update within seconds instead of waiting for the hourly sync.
						Install Jellyfin's Webhook plugin, add a Generic destination with this URL (event types: Item Added, Playback Stop,
						User Data Saved) and this template:
					</CardDescription>
				</CardHeader>
				<CardContent class="grid gap-2 text-xs">
					<code class="break-all rounded bg-muted p-2">{webhookUrl}</code>
					<code class="break-all rounded bg-muted p-2">
						{'{"NotificationType":"{{NotificationType}}","UserId":"{{UserId}}","ItemType":"{{ItemType}}"}'}
					</code>
					<p class="text-muted-foreground">The URL contains a secret token: keep it private. Jellyfin must be able to reach Tipsarr at that address.</p>
				</CardContent>
			</Card>
		{/if}
	{/if}
</div>

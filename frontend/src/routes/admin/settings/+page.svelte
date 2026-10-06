<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';

	let settings: Schemas['SettingsBody'] | null = $state(null);
	let tmdbKey = $state('');
	let message: string | null = $state(null);
	let error: string | null = $state(null);
	let saving = $state(false);

	onMount(async () => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		try {
			settings = await unwrap(api.GET('/admin/settings'));
		} catch (e) {
			error = (e as Error).message;
		}
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = message = null;
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { tmdbApiKey: tmdbKey.trim() } }));
			tmdbKey = '';
			message = 'Saved.';
		} catch (err) {
			error = (err as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Settings · Tipsarr</title>
</svelte:head>

<div class="grid max-w-xl gap-4">
	<h1 class="font-bold text-2xl">Settings</h1>

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
				<CardTitle>TMDB</CardTitle>
				<CardDescription>
					{settings.tmdbConfigured ? 'A key is saved. Enter a new one to replace it.' : 'No key yet: discover and search need one.'}
				</CardDescription>
			</CardHeader>
			<CardContent>
				<form class="grid gap-3" onsubmit={save}>
					<Input placeholder="TMDB API key or read-access token" bind:value={tmdbKey} autocomplete="off" required />
					{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
					{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}
					<Button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
				</form>
			</CardContent>
		</Card>
	{:else if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}
</div>

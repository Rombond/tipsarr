<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	let users = $state<Schemas['User'][]>([]);
	let error = $state<string | null>(null);
	let message = $state<string | null>(null);
	let busy = $state(false);

	async function load() {
		try {
			users = await unwrap(api.GET('/admin/users'));
		} catch (e) {
			error = (e as Error).message;
		}
	}

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		load();
	});

	async function patch(u: Schemas['User'], body: { role?: 'admin' | 'user'; region?: string; language?: string }) {
		error = message = null;
		try {
			const updated = await unwrap(api.PATCH('/admin/users/{id}', { params: { path: { id: u.id } }, body }));
			users = users.map((x) => (x.id === u.id ? updated : x));
		} catch (e) {
			error = (e as Error).message;
			await load(); // restore what the server has
		}
	}

	async function importUsers() {
		busy = true;
		error = message = null;
		try {
			const r = await unwrap(api.POST('/admin/users/import'));
			message = `Imported ${r.created} new user(s) (${r.total} in Jellyfin).`;
			await load();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}

	const when = (unix: number) => (unix ? new Date(unix * 1000).toLocaleDateString() : 'never');
</script>

<svelte:head>
	<title>Users · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="font-bold text-3xl">Users</h1>
		<Button variant="outline" size="sm" disabled={busy} onclick={importUsers}>Import from Jellyfin</Button>
	</div>
	<p class="text-sm text-muted-foreground">
		Users appear when they sign in with their Jellyfin account (or via import). Admins can approve requests and change settings.
	</p>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
	{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}

	<div class="grid gap-2">
		{#each users as u (u.id)}
			<div class="flex flex-wrap items-center gap-3 rounded-lg border border-border p-3 text-sm">
				<div class="min-w-32 flex-1">
					<div class="font-medium">{u.name}{#if u.id === auth.user?.id} <span class="text-xs text-muted-foreground">(you)</span>{/if}</div>
					<div class="text-xs text-muted-foreground">last sign-in {when(u.lastLoginAt)}</div>
				</div>
				<SimpleSelect
					label="Role"
					value={u.role}
					disabled={u.id === auth.user?.id}
					options={[{ value: 'user', label: 'user' }, { value: 'admin', label: 'admin' }]}
					onchange={(v) => patch(u, { role: v as 'admin' | 'user' })}
					class="h-8 min-w-24"
				/>
				<Input
					class="h-8 w-20"
					placeholder="Region"
					maxlength={2}
					value={u.region}
					aria-label="Region"
					onchange={(e) => patch(u, { region: e.currentTarget.value.trim().toUpperCase() })}
				/>
				<Input
					class="h-8 w-24"
					placeholder="Language"
					value={u.language}
					aria-label="Language"
					onchange={(e) => patch(u, { language: e.currentTarget.value.trim() })}
				/>
			</div>
		{/each}
	</div>
</div>

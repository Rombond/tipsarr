<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import ServarrForm from '$lib/components/admin/servarr-form.svelte';

	let instances = $state<Schemas['InstanceView'][]>([]);
	let error = $state<string | null>(null);
	let editing = $state<{ kind: 'radarr' | 'sonarr'; instance: Schemas['InstanceView'] | null } | null>(null);

	async function load() {
		try {
			instances = await unwrap(api.GET('/admin/servarr'));
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

	async function remove(i: Schemas['InstanceView']) {
		if (!confirm(`Remove ${i.name}? Existing requests keep their history.`)) return;
		try {
			await expectOk(api.DELETE('/admin/servarr/{id}', { params: { path: { id: i.id } } }));
			await load();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	const kinds = [
		{ kind: 'radarr', title: 'Radarr (movies)' },
		{ kind: 'sonarr', title: 'Sonarr (TV shows)' },
	] as const;
</script>

<svelte:head>
	<title>Services · Tipsarr</title>
</svelte:head>

<div class="grid max-w-2xl gap-4">
	<h1 class="font-bold text-2xl">Radarr and Sonarr</h1>
	{#if auth.dryRun}
		<p class="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
			Dry-run is ON: approving a request records it but nothing is ever sent to Radarr or Sonarr. Testing a connection only reads.
		</p>
	{/if}
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}

	{#each kinds as k (k.kind)}
		<Card>
			<CardHeader>
				<CardTitle>{k.title}</CardTitle>
				<CardDescription>The default instance receives approved requests.</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-3">
				{#each instances.filter((i) => i.kind === k.kind) as i (i.id)}
					{#if editing?.instance?.id === i.id}
						<ServarrForm kind={k.kind} instance={i} onSaved={() => { editing = null; load(); }} onCancel={() => (editing = null)} />
					{:else}
						<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3 text-sm">
							<div class="grid gap-0.5">
								<span class="flex items-center gap-2 font-medium">{i.name}{#if i.isDefault}<Badge variant="secondary">default</Badge>{/if}</span>
								<span class="font-mono text-xs text-muted-foreground">{i.url}</span>
								<span class="text-xs text-muted-foreground">profile #{i.qualityProfileId} · {i.rootFolder}</span>
							</div>
							<div class="flex gap-2">
								<Button size="sm" variant="outline" onclick={() => (editing = { kind: k.kind, instance: i })}>Edit</Button>
								<Button size="sm" variant="ghost" onclick={() => remove(i)}>Remove</Button>
							</div>
						</div>
					{/if}
				{/each}
				{#if editing && editing.instance === null && editing.kind === k.kind}
					<ServarrForm kind={k.kind} onSaved={() => { editing = null; load(); }} onCancel={() => (editing = null)} />
				{:else}
					<div>
						<Button size="sm" variant="outline" onclick={() => (editing = { kind: k.kind, instance: null })}>Add {k.kind}</Button>
					</div>
				{/if}
			</CardContent>
		</Card>
	{/each}
</div>

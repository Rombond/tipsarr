<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';

	const ALL_EVENTS = ['request.created', 'request.approved', 'request.declined', 'request.failed', 'media.available'];

	let hooks = $state<Schemas['WebhookView'][]>([]);
	let error = $state<string | null>(null);
	let message = $state<string | null>(null);
	let busy = $state(false);

	let editingId = $state<string | null>(null); // 'new' or a webhook id
	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let events = $state<string[]>([]);
	let enabled = $state(true);

	async function load() {
		try {
			hooks = await unwrap(api.GET('/admin/webhooks'));
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

	function edit(h: Schemas['WebhookView'] | null) {
		editingId = h?.id ?? 'new';
		name = h?.name ?? '';
		url = h?.url ?? '';
		secret = '';
		events = h ? [...h.events] : [];
		enabled = h?.enabled ?? true;
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = message = null;
		const body = { name: name.trim(), url: url.trim(), events, enabled, ...(secret ? { secret } : {}) };
		try {
			if (editingId === 'new') await unwrap(api.POST('/admin/webhooks', { body }));
			else await unwrap(api.PUT('/admin/webhooks/{id}', { params: { path: { id: editingId! } }, body }));
			editingId = null;
			await load();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			busy = false;
		}
	}

	async function test(h: Schemas['WebhookView']) {
		error = message = null;
		try {
			await expectOk(api.POST('/admin/webhooks/{id}/test', { params: { path: { id: h.id } } }));
			message = `Test event delivered to ${h.name}.`;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function remove(h: Schemas['WebhookView']) {
		if (!confirm(`Delete webhook ${h.name}?`)) return;
		try {
			await expectOk(api.DELETE('/admin/webhooks/{id}', { params: { path: { id: h.id } } }));
			await load();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	function toggle(ev: string) {
		events = events.includes(ev) ? events.filter((x) => x !== ev) : [...events, ev];
	}
</script>

<svelte:head>
	<title>Webhooks · Tipsarr</title>
</svelte:head>

<div class="grid max-w-2xl gap-4">
	<h1 class="font-bold text-2xl">Webhooks</h1>
	<p class="text-sm text-muted-foreground">
		Tipsarr POSTs a JSON event to each URL (works with ntfy, Discord-compatible bridges, Home Assistant…). With a secret, the body is signed:
		<code>X-Tipsarr-Signature: sha256=HMAC(secret, body)</code>. Payloads include <code>dryRun</code>.
	</p>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
	{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}

	<Card>
		<CardHeader>
			<CardTitle>Outgoing webhooks</CardTitle>
			<CardDescription>Events: {ALL_EVENTS.join(', ')}.</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-3">
			{#each hooks as h (h.id)}
				<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3 text-sm">
					<div class="grid min-w-0 gap-0.5">
						<span class="flex items-center gap-2 font-medium">
							{h.name}
							{#if !h.enabled}<Badge variant="outline">disabled</Badge>{/if}
							{#if h.secretConfigured}<Badge variant="secondary">signed</Badge>{/if}
						</span>
						<span class="truncate font-mono text-xs text-muted-foreground">{h.url}</span>
						<span class="text-xs text-muted-foreground">{h.events.length ? h.events.join(', ') : 'all events'}</span>
					</div>
					<div class="flex shrink-0 gap-2">
						<Button size="sm" variant="outline" onclick={() => test(h)}>Test</Button>
						<Button size="sm" variant="outline" onclick={() => edit(h)}>Edit</Button>
						<Button size="sm" variant="ghost" onclick={() => remove(h)}>Delete</Button>
					</div>
				</div>
			{/each}

			{#if editingId}
				<form class="grid gap-3 rounded-lg border border-border p-3" onsubmit={save}>
					<Input placeholder="Name" bind:value={name} required />
					<Input placeholder="URL (https://ntfy.sh/my-topic)" type="url" bind:value={url} required />
					<Input placeholder={editingId === 'new' ? 'Signing secret (optional)' : 'Signing secret (leave empty to keep)'} bind:value={secret} autocomplete="off" />
					<div class="grid gap-1.5 text-sm">
						<span class="text-xs text-muted-foreground">Events (none ticked = all)</span>
						{#each ALL_EVENTS as ev (ev)}
							<label class="flex items-center gap-2"><input type="checkbox" checked={events.includes(ev)} onchange={() => toggle(ev)} /> {ev}</label>
						{/each}
					</div>
					<label class="flex items-center gap-2 text-sm"><input type="checkbox" bind:checked={enabled} /> Enabled</label>
					<div class="flex gap-2">
						<Button type="submit" disabled={busy}>{busy ? 'Saving…' : 'Save'}</Button>
						<Button type="button" variant="outline" onclick={() => (editingId = null)}>Cancel</Button>
					</div>
				</form>
			{:else}
				<div><Button size="sm" variant="outline" onclick={() => edit(null)}>Add webhook</Button></div>
			{/if}
		</CardContent>
	</Card>
</div>

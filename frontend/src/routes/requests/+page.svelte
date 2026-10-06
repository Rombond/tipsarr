<script lang="ts">
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { onEvent, stream } from '$lib/events.svelte';
	import RequestRow from '$lib/components/requests/request-row.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Skeleton } from '$lib/components/ui/skeleton';

	type Filter = 'all' | 'mine' | 'pending' | 'approved' | 'available' | 'declined' | 'failed';
	const filters: Filter[] = ['all', 'pending', 'approved', 'available', 'declined', 'failed'];

	let filter = $state<Filter>('all');
	let items = $state<Schemas['View'][]>([]);
	let total = $state(0);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busyId = $state<string | null>(null);
	let declining = $state<Schemas['View'] | null>(null);
	let reason = $state('');

	async function load(showSpinner = true) {
		if (showSpinner) loading = true;
		try {
			const res = await unwrap(api.GET('/requests', { params: { query: { filter, take: 50 } } }));
			items = res.items;
			total = res.total;
			error = null;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	// refetch on filter change and after an SSE reconnect
	$effect(() => {
		filter;
		stream.reconnects;
		load();
	});

	$effect(() => {
		let timer: ReturnType<typeof setTimeout> | undefined;
		const offUpdated = onEvent('request.updated', () => {
			clearTimeout(timer);
			timer = setTimeout(() => load(false), 250); // coalesce bursts
		});
		const offProgress = onEvent('request.progress', (d: { id: string; percent: number; etaSeconds: number }) => {
			items = items.map((r) =>
				r.id === d.id ? { ...r, stage: 'downloading', progress: { percent: d.percent, etaSeconds: d.etaSeconds } } : r,
			);
		});
		return () => {
			clearTimeout(timer);
			offUpdated();
			offProgress();
		};
	});

	async function act(r: Schemas['View'], fn: () => Promise<unknown>) {
		busyId = r.id;
		error = null;
		try {
			await fn();
			await load(false);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busyId = null;
		}
	}

	const approve = (r: Schemas['View']) =>
		act(r, () =>
			r.status === 'failed'
				? unwrap(api.POST('/requests/{id}/retry', { params: { path: { id: r.id } } }))
				: unwrap(api.POST('/requests/{id}/approve', { params: { path: { id: r.id } }, body: {} })),
		);

	async function confirmDecline() {
		const r = declining;
		if (!r) return;
		declining = null;
		await act(r, () => unwrap(api.POST('/requests/{id}/decline', { params: { path: { id: r.id } }, body: { reason: reason.trim() } })));
		reason = '';
	}

	const remove = (r: Schemas['View']) => act(r, () => expectOk(api.DELETE('/requests/{id}', { params: { path: { id: r.id } } })));
</script>

<svelte:head>
	<title>Requests · Tipsarr</title>
</svelte:head>

<div class="grid max-w-3xl gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="font-bold text-2xl">Requests</h1>
		{#if auth.isAdmin}
			<div class="flex flex-wrap gap-1">
				{#each filters as f (f)}
					<Button size="sm" variant={filter === f ? 'default' : 'outline'} onclick={() => (filter = f)}>{f}</Button>
				{/each}
			</div>
		{:else}
			<p class="text-xs text-muted-foreground">Your requests</p>
		{/if}
	</div>

	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}

	{#if loading}
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-24 w-full" />
	{:else if items.length === 0}
		<p class="text-sm text-muted-foreground">No requests yet. Open a movie or show and press Request.</p>
	{:else}
		<div class="grid gap-3">
			{#each items as r (r.id)}
				<RequestRow request={r} isAdmin={auth.isAdmin} busy={busyId === r.id} onApprove={approve} onDecline={(x) => (declining = x)} onDelete={remove} />
			{/each}
		</div>
		<p class="text-xs text-muted-foreground">{total} request{total === 1 ? '' : 's'}</p>
	{/if}
</div>

<Dialog.Root open={declining !== null} onOpenChange={(o) => !o && (declining = null)}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>Decline "{declining?.title}"?</Dialog.Title>
		</Dialog.Header>
		<Input placeholder="Reason (optional, shown to the requester)" bind:value={reason} />
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (declining = null)}>Cancel</Button>
			<Button variant="destructive" onclick={confirmDecline}>Decline</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<script lang="ts">
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
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
	let tabCounts = $state<Schemas['RequestCountsResponse'] | null>(null);
	let approvingAll = $state(false);
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
			const [res, c] = await Promise.all([
				unwrap(api.GET('/requests', { params: { query: { filter, take: 50 } } })),
				unwrap(api.GET('/requests/counts')),
			]);
			items = res.items;
			total = res.total;
			tabCounts = c;
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
			toast.error(error);
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

	async function approveAll() {
		const pending = items.filter((r) => r.status === 'pending');
		if (!pending.length) return;
		const note = auth.dryRun ? ' Dry-run is on: nothing will be sent to Radarr/Sonarr.' : ' They will be sent to Radarr/Sonarr now.';
		if (!confirm(`Approve ${pending.length} pending request${pending.length === 1 ? '' : 's'}?${note}`)) return;
		approvingAll = true;
		let ok = 0;
		for (const r of pending) {
			try {
				await unwrap(api.POST('/requests/{id}/approve', { params: { path: { id: r.id } }, body: {} }));
				ok++;
			} catch (e) {
				toast.error(`${r.title}: ${(e as Error).message}`);
			}
		}
		approvingAll = false;
		toast.success(`Approved ${ok} of ${pending.length}`);
		await load(false);
	}

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

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="font-bold text-3xl">Requests</h1>
		{#if auth.isAdmin && filter === 'pending' && items.length > 1}
			<Button size="sm" disabled={approvingAll} onclick={approveAll}>{approvingAll ? 'Approving…' : `Approve all ${items.length} pending`}</Button>
		{/if}
	</div>

	<div class="flex gap-1 overflow-x-auto pb-1" role="tablist" aria-label="Request status">
		{#each filters as f (f)}
			{@const n = f === 'all' ? Object.values(tabCounts ?? {}).reduce((a, b) => a + b, 0) : ((tabCounts as Record<string, number> | null)?.[f] ?? 0)}
			<button
				type="button"
				role="tab"
				aria-selected={filter === f}
				class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-full border px-3 py-1 text-sm capitalize transition-colors {filter === f ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'}"
				onclick={() => (filter = f)}
			>
				{f}
				{#if n > 0}<span class="rounded-full bg-black/10 px-1.5 text-[11px] dark:bg-white/15 {f === 'pending' && filter !== f ? 'bg-primary text-primary-foreground dark:bg-primary' : ''}">{n}</span>{/if}
			</button>
		{/each}
	</div>

	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}

	{#if loading}
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-24 w-full" />
	{:else if items.length === 0}
		<div class="rounded-xl border border-dashed border-border p-8 text-center">
			<p class="font-medium">{filter === 'all' ? 'No requests yet' : `No ${filter} requests`}</p>
			<p class="mt-1 text-sm text-muted-foreground">Find something on <a class="underline" href="/discover">Discover</a> or the <a class="underline" href="/boxoffice">box office</a> and press Request.</p>
		</div>
	{:else}
		<div class="grid gap-3 xl:grid-cols-2">
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

<script lang="ts">
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';

	let {
		type,
		tmdbId,
		seasons = [],
		availability,
		requestStatus,
		onRequested,
		title = '',
	}: {
		type: 'movie' | 'tv';
		tmdbId: number;
		seasons?: Schemas['Season'][];
		availability: string;
		requestStatus?: string;
		onRequested?: (r: Schemas['View']) => void;
		title?: string;
	} = $props();

	let created = $state<string | null>(null);
	let busy = $state(false);
	let error = $state<string | null>(null);
	let pickerOpen = $state(false);
	let picked = $state<number[]>([]);

	const regular = $derived(seasons.filter((s) => s.number > 0));
	const status = $derived(created ?? requestStatus ?? null);

	const label = $derived(
		availability === 'available'
			? 'Available'
			: status === 'pending'
				? 'Requested'
				: status === 'approved'
					? 'Approved'
					: 'Request',
	);
	const disabled = $derived(busy || availability === 'available' || status !== null);

	async function submit(chosen: number[]) {
		busy = true;
		error = null;
		try {
			const r = await unwrap(
				api.POST('/requests', { body: { type, tmdbId, ...(type === 'tv' ? { seasons: chosen } : {}) } }),
			);
			created = r.status;
			pickerOpen = false;
			toast.success(r.status === 'approved' ? `${title ? `"${title}"` : 'Request'} approved${r.dryRun ? ' (dry-run: nothing sent)' : ''}` : `Requested${title ? ` "${title}"` : ''}`);
			onRequested?.(r);
		} catch (e) {
			error = (e as Error).message;
			toast.error(error);
		} finally {
			busy = false;
		}
	}

	function click() {
		if (type === 'tv' && regular.length > 1) {
			picked = regular.map((s) => s.number);
			pickerOpen = true;
		} else {
			submit(regular.map((s) => s.number));
		}
	}

	function toggle(n: number) {
		picked = picked.includes(n) ? picked.filter((x) => x !== n) : [...picked, n].sort((a, b) => a - b);
	}
</script>

<div class="grid gap-1.5">
	<div>
		<Button {disabled} onclick={click}>{busy && !pickerOpen ? 'Requesting…' : label}</Button>
	</div>
	{#if error && !pickerOpen}<p class="text-sm text-destructive">{error}</p>{/if}
</div>

<Dialog.Root bind:open={pickerOpen}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>Which seasons?</Dialog.Title>
		</Dialog.Header>
		<div class="grid max-h-72 gap-1.5 overflow-y-auto text-sm">
			<button
				type="button"
				class="w-fit cursor-pointer text-xs underline"
				onclick={() => (picked = picked.length === regular.length ? [] : regular.map((s) => s.number))}
			>
				{picked.length === regular.length ? 'Select none' : 'Select all'}
			</button>
			{#each regular as s (s.number)}
				<label class="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-2">
					<input type="checkbox" checked={picked.includes(s.number)} onchange={() => toggle(s.number)} />
					<span class="font-medium">{s.name || `Season ${s.number}`}</span>
					<span class="ml-auto text-muted-foreground">{s.episodeCount} episodes</span>
				</label>
			{/each}
		</div>
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (pickerOpen = false)}>Cancel</Button>
			<Button disabled={busy || picked.length === 0} onclick={() => submit(picked)}>
				{busy ? 'Requesting…' : `Request ${picked.length} season${picked.length === 1 ? '' : 's'}`}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

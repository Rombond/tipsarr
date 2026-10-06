<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import type { Schemas } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';

	let {
		open = $bindable(false),
		seasons,
		busy = false,
		error = null,
		onsubmit,
	}: {
		open?: boolean;
		seasons: Schemas['Season'][];
		busy?: boolean;
		error?: string | null;
		onsubmit: (chosen: number[]) => void;
	} = $props();

	const regular = $derived(seasons.filter((s) => s.number > 0));
	let picked = $state<number[]>([]);

	// every season starts selected each time the dialog opens
	$effect(() => {
		if (open) picked = regular.map((s) => s.number);
	});

	function toggle(n: number) {
		picked = picked.includes(n) ? picked.filter((x) => x !== n) : [...picked, n].sort((a, b) => a - b);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{t('req.which_seasons')}</Dialog.Title>
		</Dialog.Header>
		<div class="grid max-h-72 gap-1.5 overflow-y-auto text-sm">
			<button
				type="button"
				class="w-fit cursor-pointer text-xs underline"
				onclick={() => (picked = picked.length === regular.length ? [] : regular.map((s) => s.number))}
			>
				{picked.length === regular.length ? t('req.select_none') : t('req.select_all')}
			</button>
			{#each regular as s (s.number)}
				<label class="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-2">
					<input type="checkbox" checked={picked.includes(s.number)} onchange={() => toggle(s.number)} />
					<span class="font-medium">{s.name || t('seasons.season', { n: s.number })}</span>
					<span class="ml-auto text-muted-foreground">{t('seasons.episodes', { count: s.episodeCount })}</span>
				</label>
			{/each}
		</div>
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button disabled={busy || picked.length === 0} onclick={() => onsubmit(picked)}>
				{busy ? t('media.requesting') : t('req.request_seasons', { count: picked.length })}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

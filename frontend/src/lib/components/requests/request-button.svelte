<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import SeasonPicker from './season-picker.svelte';

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

	const regular = $derived(seasons.filter((s) => s.number > 0));
	const status = $derived(created ?? requestStatus ?? null);

	const label = $derived(
		availability === 'available'
			? t('media.available')
			: status === 'pending'
				? t('media.requested')
				: status === 'approved'
					? t('media.approved')
					: t('media.request'),
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
			toast.success(r.status === 'approved' ? (title ? t(r.dryRun ? 'media.toast_approved_dry' : 'media.toast_approved', { title }) : t('req.toast_approved_generic')) : title ? t('media.toast_requested', { title }) : t('req.toast_requested_generic'));
			onRequested?.(r);
		} catch (e) {
			error = errorText(e);
			toast.error(error);
		} finally {
			busy = false;
		}
	}

	function click() {
		if (type === 'tv' && regular.length > 1) {
			pickerOpen = true;
		} else {
			submit(regular.map((s) => s.number));
		}
	}
</script>

<div class="grid gap-1.5">
	<div>
		<Button {disabled} onclick={click}>{busy && !pickerOpen ? t('media.requesting') : label}</Button>
	</div>
	{#if error && !pickerOpen}<p class="text-sm text-destructive">{error}</p>{/if}
</div>

<SeasonPicker bind:open={pickerOpen} seasons={regular} {busy} {error} onsubmit={submit} />

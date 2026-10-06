<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { type Schemas } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import RequestFlow from './request-flow.svelte';

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
	let flow: { start: () => void } | undefined = $state();

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
</script>

<Button {disabled} onclick={() => flow?.start()}>{busy ? t('media.requesting') : label}</Button>
<RequestFlow
	bind:this={flow}
	bind:busy
	{type}
	{tmdbId}
	{title}
	{seasons}
	onrequested={(r) => {
		created = r.status;
		onRequested?.(r);
	}}
/>

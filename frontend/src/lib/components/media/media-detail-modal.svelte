<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, imageUrl, errorText, type MediaItem } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import StarIcon from '@lucide/svelte/icons/star';

	let {
		open = false,
		item = null,
		onclose,
	}: { open?: boolean; item?: MediaItem | null; onclose?: () => void } = $props();

	let requested = $state<string | null>(null);
	let busy = $state(false);

	$effect(() => {
		if (open) requested = null;
	});

	const status = $derived(requested ?? item?.requestStatus ?? null);
	const backdrop = $derived(imageUrl(item?.backdropPath, 'w780'));

	function handleOpenChange(next: boolean) {
		if (!next) onclose?.();
	}

	function handleViewDetails() {
		if (!item) return;
		const url = `/media/${item.type}/${item.tmdbId}`;
		onclose?.();
		goto(url);
	}

	async function request() {
		if (!item) return;
		if (item.type === 'tv') return handleViewDetails(); // seasons are chosen on the details page
		busy = true;
		try {
			const r = await unwrap(api.POST('/requests', { body: { type: item.type, tmdbId: item.tmdbId } }));
			requested = r.status;
			toast.success(r.status === 'approved' ? t(r.dryRun ? 'media.toast_approved_dry' : 'media.toast_approved', { title: item.title }) : t('media.toast_requested', { title: item.title }));
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
	<Dialog.Content class="overflow-hidden p-0 sm:max-w-lg">
		{#if item}
			{#if backdrop}
				<div class="relative h-40 w-full">
					<img src={backdrop} alt="" class="h-full w-full object-cover" />
					<div class="absolute inset-0 bg-gradient-to-t from-popover to-transparent"></div>
				</div>
			{/if}
			<div class="grid gap-3 p-5 {backdrop ? '-mt-10 relative' : ''}">
				<Dialog.Header>
					<div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
						<Badge variant="secondary">{item.type === 'tv' ? t('type.tv_short') : t('type.movie')}</Badge>
						{#if item.releaseDate}<span>{item.releaseDate.slice(0, 4)}</span>{/if}
						{#if item.voteAverage}<span class="inline-flex items-center gap-0.5"><StarIcon class="size-3 fill-amber-400 text-amber-400" />{item.voteAverage.toFixed(1)}</span>{/if}
						{#if item.availability !== 'none'}<Badge>{item.availability === 'available' ? t('media.available') : t('card.partial')}</Badge>{/if}
					</div>
					<Dialog.Title class="text-xl">{item.title}</Dialog.Title>
				</Dialog.Header>
				{#if item.overview}<p class="line-clamp-6 text-sm text-muted-foreground">{item.overview}</p>{/if}
				<Dialog.Footer class="gap-2">
					<Button variant="outline" onclick={handleViewDetails}>{t('modal.more_details')}</Button>
					{#if item.availability !== 'available'}
						<Button disabled={busy || status !== null} onclick={request}>
							{status === 'pending' ? t('media.requested') : status === 'approved' ? t('media.approved') : item.type === 'tv' ? t('hero.choose_seasons') : busy ? t('media.requesting') : t('media.request')}
						</Button>
					{/if}
				</Dialog.Footer>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

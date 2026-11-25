<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, imageUrl, type MediaItem } from '$lib/api/client';
	import * as Dialog from '$lib/components/ui/dialog';
	import RequestFlow from '$lib/components/requests/request-flow.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import StatusIcon from '$lib/components/ui/status-icon.svelte';
	import { itemStatus } from '$lib/status';
	import { Button } from '$lib/components/ui/button';
	import StarIcon from '@lucide/svelte/icons/star';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import BookmarkCheckIcon from '@lucide/svelte/icons/bookmark-check';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import EyeIcon from '@lucide/svelte/icons/eye';
	import { untrack } from 'svelte';
	import { MediaFlags } from '$lib/media-flags.svelte';

	let {
		open = false,
		item = null,
		onclose,
	}: { open?: boolean; item?: MediaItem | null; onclose?: () => void } = $props();

	let requested = $state<string | null>(null);
	// list results often lack an overview in the person's language: the details carry the English fallback
	let fetchedOverview = $state('');
	let busy = $state(false);
	let flow: { start: () => void } | undefined = $state();
	// watchlist / hidden flags of the open title, same logic as the detail page
	const flags = $derived(item ? new MediaFlags(item.type, item.tmdbId) : null);

	$effect(() => {
		if (!open) return;
		requested = null;
		fetchedOverview = '';
		const it = item;
		const f = flags;
		if (f) untrack(() => f.load());
		if (it && !it.overview) {
			unwrap(api.GET('/media/{type}/{id}', { params: { path: { type: it.type, id: it.tmdbId } } }))
				.then((d) => (fetchedOverview = d.overview ?? ''))
				.catch(() => {});
		}
	});

	const status = $derived(requested ?? item?.requestStatus ?? null);
	const backdrop = $derived(imageUrl(item?.backdropPath, 'w780'));

	function handleOpenChange(next: boolean) {
		if (!next) onclose?.();
	}

	// hiding closes the popup; Discover lists drop the title through `hiddenTitles`
	async function toggleHidden() {
		const f = flags;
		if (!f) return;
		const hiding = !f.blocklisted;
		if ((await f.toggle('blocklist')) && hiding) onclose?.();
	}

	const detailsUrl = $derived(item ? `/media/${item.type}/${item.tmdbId}` : '#');
	const shown = $derived(item ? itemStatus({ availability: item.availability, requestStatus: status }) : null);

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
						{#if shown}<StatusIcon status={shown} />{/if}
					</div>
					<Dialog.Title class="text-xl">{item.title}</Dialog.Title>
				</Dialog.Header>
				{#if item.overview || fetchedOverview}<p class="line-clamp-6 text-sm text-muted-foreground">{item.overview || fetchedOverview}</p>{/if}
				<Dialog.Footer class="gap-2">
					{#if flags}
						<div class="flex gap-2 sm:mr-auto">
							<Button
								variant="outline"
								size="icon"
								disabled={flags.busy}
								aria-pressed={flags.watchlisted}
								aria-label={flags.watchlisted ? t('actions.on_watchlist') : t('actions.watchlist')}
								title={flags.watchlisted ? t('actions.on_watchlist') : t('actions.watchlist')}
								onclick={() => flags.toggle('watchlist')}
							>
								{#if flags.watchlisted}<BookmarkCheckIcon class="size-4" />{:else}<BookmarkIcon class="size-4" />{/if}
							</Button>
							<Button
								variant="outline"
								size="icon"
								disabled={flags.busy}
								aria-label={flags.blocklisted ? t('actions.show_again') : t('actions.hide')}
								title={flags.blocklisted ? t('actions.show_again') : t('actions.hide')}
								onclick={toggleHidden}
							>
								{#if flags.blocklisted}<EyeIcon class="size-4" />{:else}<EyeOffIcon class="size-4" />{/if}
							</Button>
						</div>
					{/if}
					<Button variant="outline" href={detailsUrl} onclick={() => onclose?.()}>{t('modal.more_details')}</Button>
					{#if item.availability !== 'available' && status === null}
						<Button disabled={busy} onclick={() => flow?.start()}>{busy ? t('media.requesting') : t('media.request')}</Button>
					{/if}
				</Dialog.Footer>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

{#if item}
	<RequestFlow bind:this={flow} bind:busy type={item.type} tmdbId={item.tmdbId} title={item.title} onrequested={(r) => (requested = r.status)} />
{/if}

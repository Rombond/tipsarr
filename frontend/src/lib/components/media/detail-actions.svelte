<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, expectOk, errorText, type MediaDetail } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import RequestButton from '$lib/components/requests/request-button.svelte';
	import PlayIcon from '@lucide/svelte/icons/play';
	import ClapperboardIcon from '@lucide/svelte/icons/clapperboard';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import BookmarkCheckIcon from '@lucide/svelte/icons/bookmark-check';
	import MoreIcon from '@lucide/svelte/icons/ellipsis';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import EyeIcon from '@lucide/svelte/icons/eye';
	import ExternalIcon from '@lucide/svelte/icons/external-link';

	let { details, type, tmdbId }: { details: MediaDetail; type: 'movie' | 'tv'; tmdbId: number } = $props();

	let watchlisted = $state(false);
	let blocklisted = $state(false);
	let busy = $state(false);

	$effect(() => {
		const t = type;
		const id = tmdbId;
		unwrap(api.GET('/media/{type}/{id}/flags', { params: { path: { type: t, id } } }))
			.then((f) => {
				watchlisted = f.watchlisted;
				blocklisted = f.blocklisted;
			})
			.catch(() => {});
	});

	async function toggle(kind: 'watchlist' | 'blocklist') {
		busy = true;
		const on = kind === 'watchlist' ? watchlisted : blocklisted;
		try {
			if (kind === 'watchlist') {
				if (on) await expectOk(api.DELETE('/watchlist/{type}/{id}', { params: { path: { type, id: tmdbId } } }));
				else await expectOk(api.POST('/watchlist', { body: { type, tmdbId } }));
				watchlisted = !on;
				toast.success(t(on ? 'actions.toast_watch_removed' : 'actions.toast_watch_added'));
			} else {
				if (on) await expectOk(api.DELETE('/blocklist/{type}/{id}', { params: { path: { type, id: tmdbId } } }));
				else await expectOk(api.POST('/blocklist', { body: { type, tmdbId } }));
				blocklisted = !on;
				toast.success(t(on ? 'actions.toast_unhidden' : 'actions.toast_hidden'));
			}
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			busy = false;
		}
	}
</script>

<div class="flex flex-wrap items-center gap-2">
	<RequestButton
		{type}
		{tmdbId}
		seasons={details.seasons}
		availability={details.availability}
		requestStatus={details.requestStatus}
		title={details.title}
	/>

	{#if details.watchUrl}
		<Button href={details.watchUrl} target="_blank" rel="noreferrer" class="bg-emerald-600 text-white hover:bg-emerald-600/90">
			<PlayIcon class="size-4" />
			{t('actions.play')}
		</Button>
	{/if}

	{#if details.trailerKey}
		<Button variant="outline" href="https://www.youtube.com/watch?v={details.trailerKey}" target="_blank" rel="noreferrer">
			<ClapperboardIcon class="size-4" />
			{t('actions.trailer')}
		</Button>
	{/if}

	<Button variant="outline" disabled={busy} onclick={() => toggle('watchlist')} aria-pressed={watchlisted}>
		{#if watchlisted}<BookmarkCheckIcon class="size-4" /> {t('actions.on_watchlist')}{:else}<BookmarkIcon class="size-4" /> {t('actions.watchlist')}{/if}
	</Button>

	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button variant="ghost" size="icon" aria-label={t('common.more_actions')} {...props}><MoreIcon class="size-5" /></Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="start" class="w-64">
			<DropdownMenu.Item onclick={() => toggle('blocklist')}>
				{#if blocklisted}<EyeIcon /> {t('actions.show_again')}{:else}<EyeOffIcon /> {t('actions.hide')}{/if}
			</DropdownMenu.Item>
			<DropdownMenu.Separator />
			{#if details.imdbId}
				<DropdownMenu.Item onclick={() => window.open(`https://www.imdb.com/title/${details.imdbId}`, '_blank', 'noreferrer')}>
					<ExternalIcon /> {t('actions.imdb')}
				</DropdownMenu.Item>
			{/if}
			<DropdownMenu.Item onclick={() => window.open(`https://www.themoviedb.org/${type}/${tmdbId}`, '_blank', 'noreferrer')}>
				<ExternalIcon /> {t('actions.tmdb')}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</div>

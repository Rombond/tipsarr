<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { untrack } from 'svelte';
	import type { MediaDetail } from '$lib/api/client';
	import { MediaFlags } from '$lib/media-flags.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { auth } from '$lib/stores/auth.svelte';
	import ReportIssueDialog from '$lib/components/issues/report-issue-dialog.svelte';
	import FlagIcon from '@lucide/svelte/icons/flag';
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

	let reporting = $state(false);

	// recreated when the page moves to another title
	const flags = $derived(new MediaFlags(type, tmdbId));
	$effect(() => {
		const f = flags;
		untrack(() => f.load());
	});
</script>

<div class="flex flex-wrap items-center gap-2">
	{#if auth.mayRequest(type)}
	<RequestButton
		{type}
		{tmdbId}
		seasons={details.seasons}
		availability={details.availability}
		requestStatus={details.requestStatus}
		monitored={details.monitored}
		title={details.title}
		seasonNumbering={details.seasonNumbering}
		numberingDiffers={details.seasonNumberingDiffers}
	/>
	{/if}

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

	<Button variant="outline" disabled={flags.busy} onclick={() => flags.toggle('watchlist')} aria-pressed={flags.watchlisted}>
		{#if flags.watchlisted}<BookmarkCheckIcon class="size-4" /> {t('actions.on_watchlist')}{:else}<BookmarkIcon class="size-4" /> {t('actions.watchlist')}{/if}
	</Button>

	{#if auth.canCreateIssues && details.availability !== 'none'}
		<Button variant="outline" onclick={() => (reporting = true)}>
			<FlagIcon class="size-4" />
			{t('actions.report_short')}
		</Button>
	{/if}

	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button variant="ghost" size="icon" aria-label={t('common.more_actions')} {...props}><MoreIcon class="size-5" /></Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="start" class="w-64">
			<DropdownMenu.Item onclick={() => flags.toggle('blocklist')}>
				{#if flags.blocklisted}<EyeIcon /> {t('actions.show_again')}{:else}<EyeOffIcon /> {t('actions.hide')}{/if}
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

<ReportIssueDialog bind:open={reporting} {type} {tmdbId} title={details.title} seasons={details.seasons ?? []} />

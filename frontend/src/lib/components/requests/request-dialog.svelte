<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import type { Schemas } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	export type RequestChoice = { seasons?: number[]; qualityProfileId?: number; rootFolder?: string };

	let {
		open = $bindable(false),
		type,
		title = '',
		seasons = [],
		options = null,
		busy = false,
		error = null,
		onsubmit,
	}: {
		open?: boolean;
		type: 'movie' | 'tv';
		title?: string;
		seasons?: Schemas['Season'][];
		/** Quality profiles and folders of the default Radarr/Sonarr; null when none is configured. */
		options?: Schemas['Options'] | null;
		busy?: boolean;
		error?: string | null;
		onsubmit: (choice: RequestChoice) => void;
	} = $props();

	const regular = $derived(seasons.filter((s) => s.number > 0));
	const askSeasons = $derived(type === 'tv' && regular.length > 1);
	let picked = $state<number[]>([]);
	let profile = $state('');
	let folder = $state('');

	// every season starts selected, and the profile/folder start at the instance defaults
	$effect(() => {
		if (!open) return;
		picked = regular.map((s) => s.number);
		profile = options ? String(options.qualityProfileId) : '';
		folder = options?.rootFolder ?? '';
	});

	function toggle(n: number) {
		picked = picked.includes(n) ? picked.filter((x) => x !== n) : [...picked, n].sort((a, b) => a - b);
	}

	function submit() {
		const choice: RequestChoice = {};
		if (type === 'tv') choice.seasons = askSeasons ? picked : regular.map((s) => s.number);
		if (options) {
			if (Number(profile) !== options.qualityProfileId) choice.qualityProfileId = Number(profile);
			if (options.rootFolders.length && folder !== options.rootFolder) choice.rootFolder = folder;
		}
		onsubmit(choice);
	}

	const profileOptions = $derived((options?.profiles ?? []).map((p) => ({ value: String(p.id), label: p.name })));
	const folderOptions = $derived((options?.rootFolders ?? []).map((f) => ({ value: f.path, label: f.path })));
</script>

<Dialog.Root bind:open>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{title ? t('req.dialog_title', { title }) : t('req.which_seasons')}</Dialog.Title>
		</Dialog.Header>

		{#if askSeasons}
			<div class="grid max-h-72 gap-1.5 overflow-y-auto text-sm">
				<div class="flex items-center justify-between">
					<span class="font-medium">{t('req.which_seasons')}</span>
					<button
						type="button"
						class="cursor-pointer text-xs underline"
						onclick={() => (picked = picked.length === regular.length ? [] : regular.map((s) => s.number))}
					>
						{picked.length === regular.length ? t('req.select_none') : t('req.select_all')}
					</button>
				</div>
				{#each regular as s (s.number)}
					<label class="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-2">
						<input type="checkbox" checked={picked.includes(s.number)} onchange={() => toggle(s.number)} />
						<span class="font-medium">{s.name || t('seasons.season', { n: s.number })}</span>
						<span class="ml-auto text-muted-foreground">{t('seasons.episodes', { count: s.episodeCount })}</span>
					</label>
				{/each}
			</div>
		{/if}

		{#if options}
			<div class="grid gap-3 rounded-lg border border-border bg-muted/40 p-3 text-sm">
				<p class="text-xs text-muted-foreground">{t('req.options_hint', { name: options.instanceName })}</p>
				<div class="grid gap-1.5">
					<span class="font-medium">{t('req.quality_profile')}</span>
					<SimpleSelect label={t('req.quality_profile')} value={profile} options={profileOptions} onchange={(v) => (profile = v)} class="w-full" />
				</div>
				{#if options.rootFolders.length}
					<div class="grid gap-1.5">
						<span class="font-medium">{t('req.root_folder')}</span>
						<SimpleSelect label={t('req.root_folder')} value={folder} options={folderOptions} onchange={(v) => (folder = v)} class="w-full" />
					</div>
				{/if}
			</div>
		{/if}

		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button disabled={busy || (askSeasons && picked.length === 0)} onclick={submit}>
				{busy ? t('media.requesting') : askSeasons ? t('req.request_seasons', { count: picked.length }) : t('media.request')}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

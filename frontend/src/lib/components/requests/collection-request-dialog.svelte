<script lang="ts">
	// "Request missing" on a collection: pick the parts, optionally a quality profile, then one normal
	// request per part on the server (same rights, limit and approval rules as a single request).
	import { untrack } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, imageUrl, ApiError, type MediaItem, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	type Skip = Schemas['CollectionSkip'];

	let {
		open = $bindable(false),
		collectionId,
		name,
		parts,
		onrequested,
	}: {
		open?: boolean;
		collectionId: number;
		name: string;
		/** Parts that can still be requested (not in the library, no active request). */
		parts: MediaItem[];
		/** After the server answered (some parts may have been requested): reload the collection. */
		onrequested?: () => void;
	} = $props();

	const today = new Date().toISOString().slice(0, 10);
	const released = (it: MediaItem) => !!it.releaseDate && it.releaseDate <= today;

	let picked = $state<number[]>([]);
	let options = $state<Schemas['Options'] | null>(null);
	let quota = $state<Schemas['Quota'] | null>(null);
	let profile = $state('');
	let folder = $state('');
	let busy = $state(false);
	let error = $state<string | null>(null);
	let skipped = $state<Skip[]>([]);

	const quotaText = $derived.by(() => {
		const w = quota?.movie;
		if (!w || !w.limit) return '';
		if (w.restricted) return t('req.quota_none', { date: w.resetsAt ? fmtDate(w.resetsAt) : '' });
		return t('req.quota_movie', { remaining: w.remaining ?? 0, days: w.days, count: w.remaining ?? 0 });
	});

	// on open: released parts start selected (announced films only when picked), options and quota are read
	// (only `open` is tracked: the parts change when the page reloads after a request, the result must stay)
	$effect(() => {
		if (!open) return;
		untrack(() => {
			picked = parts.filter(released).map((p) => p.tmdbId);
			error = null;
			skipped = [];
			options = null;
			quota = null;
		});
		unwrap(api.GET('/requests/options', { params: { query: { type: 'movie' } } }))
			.then((o) => {
				options = o;
				profile = String(o.qualityProfileId);
				folder = o.rootFolder ?? '';
			})
			.catch(() => {});
		unwrap(api.GET('/me/quota'))
			.then((q) => (quota = q))
			.catch(() => {});
	});

	function toggle(id: number) {
		picked = picked.includes(id) ? picked.filter((x) => x !== id) : [...picked, id];
	}

	const profileOptions = $derived((options?.profiles ?? []).map((p) => ({ value: String(p.id), label: p.uhd ? t('req.profile_uhd', { name: p.name }) : p.name })));
	const folderOptions = $derived((options?.rootFolders ?? []).map((f) => ({ value: f.path, label: f.path })));

	async function submit() {
		busy = true;
		error = null;
		const body: { tmdbIds: number[]; qualityProfileId?: number; rootFolder?: string } = { tmdbIds: parts.filter((p) => picked.includes(p.tmdbId)).map((p) => p.tmdbId) };
		if (options) {
			if (Number(profile) !== options.qualityProfileId) body.qualityProfileId = Number(profile);
			if (options.rootFolders.length && folder !== options.rootFolder) body.rootFolder = folder;
		}
		try {
			const r = await unwrap(api.POST('/collection/{id}/request', { params: { path: { id: collectionId } }, body }));
			if (r.requested.length) toast.success(t('collection.toast_requested', { count: r.requested.length }));
			else toast.info(t('collection.toast_none'));
			onrequested?.();
			unwrap(api.GET('/me/quota'))
				.then((q) => (quota = q))
				.catch(() => {});
			// keep the dialog open to say why some parts were not requested
			skipped = r.skipped;
			if (!skipped.length) open = false;
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	const reason = (s: Skip) => errorText(new ApiError(s.message, 0, s.code));
</script>

<Dialog.Root bind:open>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{t('collection.dialog_title', { name })}</Dialog.Title>
			<Dialog.Description>{t('collection.dialog_desc')}</Dialog.Description>
			{#if quotaText}<Dialog.Description>{quotaText}</Dialog.Description>{/if}
		</Dialog.Header>

		{#if skipped.length}
			<div class="grid gap-1.5 text-sm" role="status">
				<span class="font-medium">{t('collection.skipped_title')}</span>
				<ul class="grid max-h-72 gap-1 overflow-y-auto">
					{#each skipped as s (s.tmdbId)}
						<li class="rounded-md border border-border px-3 py-2">
							<span class="font-medium">{s.title || s.tmdbId}</span>
							<span class="block text-xs text-muted-foreground">{reason(s)}</span>
						</li>
					{/each}
				</ul>
			</div>
			<Dialog.Footer>
				<Button onclick={() => (open = false)}>{t('collection.close')}</Button>
			</Dialog.Footer>
		{:else}
			<div class="grid max-h-72 gap-1.5 overflow-y-auto text-sm">
				<div class="flex items-center justify-between">
					<span class="font-medium">{t('collection.movies')}</span>
					<button
						type="button"
						class="cursor-pointer text-xs underline"
						onclick={() => (picked = picked.length === parts.length ? [] : parts.map((p) => p.tmdbId))}
					>
						{picked.length === parts.length ? t('req.select_none') : t('req.select_all')}
					</button>
				</div>
				{#each parts as p (p.tmdbId)}
					{@const poster = imageUrl(p.posterPath, 'w92')}
					<label class="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-2">
						<input type="checkbox" checked={picked.includes(p.tmdbId)} onchange={() => toggle(p.tmdbId)} />
						{#if poster}<img src={poster} alt="" class="h-9 w-6 shrink-0 rounded-sm object-cover" loading="lazy" />{/if}
						<span class="min-w-0 truncate font-medium">{p.title}</span>
						<span class="ml-auto shrink-0 text-xs text-muted-foreground">
							{released(p) ? p.releaseDate?.slice(0, 4) : t('collection.unreleased')}
						</span>
					</label>
				{/each}
			</div>

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

			{#if error}<p class="text-sm text-destructive" role="alert">{error}</p>{/if}
			<Dialog.Footer>
				<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
				<Button disabled={busy || picked.length === 0} onclick={submit}>
					{busy ? t('media.requesting') : t('collection.request_n', { count: picked.length })}
				</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>

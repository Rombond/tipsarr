<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	let {
		open = $bindable(false),
		request,
		onsaved,
	}: { open?: boolean; request: Schemas['View']; onsaved?: (r: Schemas['View']) => void } = $props();

	let options = $state<Schemas['Options'] | null>(null);
	let profile = $state('');
	let folder = $state('');
	let busy = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (!open) return;
		options = null;
		error = null;
		unwrap(api.GET('/requests/options', { params: { query: { type: request.type } } }))
			.then((o) => {
				options = o;
				profile = String(request.qualityProfileId || o.qualityProfileId);
				folder = request.rootFolder || o.rootFolder;
			})
			.catch((e) => (error = errorText(e)));
	});

	async function save() {
		if (!options) return;
		busy = true;
		error = null;
		try {
			const r = await unwrap(
				api.PATCH('/requests/{id}', {
					params: { path: { id: request.id } },
					body: {
						...(Number(profile) !== options.qualityProfileId ? { qualityProfileId: Number(profile) } : {}),
						...(options.rootFolders.length && folder !== options.rootFolder ? { rootFolder: folder } : {}),
					},
				}),
			);
			open = false;
			toast.success(t('common.saved'));
			onsaved?.(r);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{t('req.change_options', { title: request.title })}</Dialog.Title>
		</Dialog.Header>
		{#if options}
			<div class="grid gap-3 text-sm">
				<div class="grid gap-1.5">
					<span class="font-medium">{t('req.quality_profile')}</span>
					<SimpleSelect label={t('req.quality_profile')} value={profile} options={options.profiles.map((p) => ({ value: String(p.id), label: p.name }))} onchange={(v) => (profile = v)} class="w-full" />
				</div>
				{#if options.rootFolders.length}
					<div class="grid gap-1.5">
						<span class="font-medium">{t('req.root_folder')}</span>
						<SimpleSelect label={t('req.root_folder')} value={folder} options={options.rootFolders.map((f) => ({ value: f.path, label: f.path }))} onchange={(v) => (folder = v)} class="w-full" />
					</div>
				{/if}
			</div>
		{/if}
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button disabled={busy || !options} onclick={save}>{busy ? t('common.saving') : t('common.save')}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

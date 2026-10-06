<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	let {
		kind,
		instance = null,
		onSaved,
		onCancel,
	}: {
		kind: 'radarr' | 'sonarr';
		instance?: Schemas['InstanceView'] | null;
		onSaved: () => void;
		onCancel: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(instance?.name ?? (kind === 'radarr' ? 'Radarr' : 'Sonarr'));
	// svelte-ignore state_referenced_locally
	let url = $state(instance?.url ?? '');
	let apiKey = $state('');
	// svelte-ignore state_referenced_locally
	let profileId = $state<number>(instance?.qualityProfileId ?? 0);
	// svelte-ignore state_referenced_locally
	let rootFolder = $state(instance?.rootFolder ?? '');
	// svelte-ignore state_referenced_locally
	let isDefault = $state(instance?.isDefault ?? false);
	// svelte-ignore state_referenced_locally
	let animeRoot = $state(instance?.animeRoot ?? '');
	let probe = $state<Schemas['ProbeResult'] | null>(null);
	let status = $state<string | null>(null);
	let error = $state<string | null>(null);
	let busy = $state(false);

	async function test() {
		busy = true;
		error = status = null;
		try {
			probe = await unwrap(api.POST('/admin/servarr/probe', { body: { kind, url: url.trim(), apiKey: apiKey.trim(), id: instance?.id } }));
			status = t('form.connected', { app: probe.appName, version: probe.version });
			if (!probe.profiles.some((p) => p.id === profileId)) profileId = probe.profiles[0]?.id ?? 0;
			if (!probe.rootFolders.some((f) => f.path === rootFolder)) rootFolder = probe.rootFolders[0]?.path ?? '';
		} catch (e) {
			probe = null;
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			const body = { kind, name: name.trim(), url: url.trim(), apiKey: apiKey.trim(), qualityProfileId: profileId, rootFolder, isDefault, ...(kind === 'sonarr' ? { animeRoot } : {}) };
			if (instance) await unwrap(api.PUT('/admin/servarr/{id}', { params: { path: { id: instance.id } }, body }));
			else await unwrap(api.POST('/admin/servarr', { body }));
			onSaved();
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}

</script>

<form class="grid gap-3 rounded-lg border border-border p-3" onsubmit={save}>
	<Input placeholder={t('form.name')} bind:value={name} required />
	<Input placeholder={t('form.url', { port: kind === 'radarr' ? 7878 : 8989 })} type="url" bind:value={url} required />
	<Input
		placeholder={instance ? t('form.api_key_keep') : t('form.api_key')}
		bind:value={apiKey}
		autocomplete="off"
		required={!instance}
	/>
	<div class="flex items-center gap-2">
		<Button type="button" variant="outline" size="sm" disabled={busy || !url.trim() || (!apiKey.trim() && !instance)} onclick={test}>
			{t('form.test')}
		</Button>
		{#if status}<span class="text-xs text-muted-foreground">{status}</span>{/if}
	</div>

	{#if probe}
		<div class="grid gap-1 text-xs">
			{t('form.profile')}
			<SimpleSelect label={t('form.profile')} value={String(profileId)} options={probe.profiles.map((p) => ({ value: String(p.id), label: p.name }))} onchange={(v) => (profileId = Number(v))} class="w-full" />
		</div>
		<div class="grid gap-1 text-xs">
			{t('form.root')}
			<SimpleSelect label={t('form.root')} value={rootFolder} options={probe.rootFolders.map((f) => ({ value: f.path, label: f.path }))} onchange={(v) => (rootFolder = v)} class="w-full" />
		</div>
		{#if kind === 'sonarr'}
			<div class="grid gap-1 text-xs">
				{t('form.anime')}
				<SimpleSelect label={t('form.root')} value={animeRoot} options={[{ value: '', label: t('form.anime_same') }, ...probe.rootFolders.map((f) => ({ value: f.path, label: f.path }))]} onchange={(v) => (animeRoot = v)} class="w-full" />
			</div>
		{/if}
	{:else if instance}
		<p class="text-xs text-muted-foreground">
			{t('services.profile_line', { id: instance.qualityProfileId, folder: instance.rootFolder })}
			{#if kind === 'sonarr' && instance.animeRoot}· {t('form.anime_summary', { folder: instance.animeRoot })}{/if}. {t('form.change_hint')}
		</p>
	{/if}

	<label class="flex items-center gap-2 text-sm">
		<input type="checkbox" bind:checked={isDefault} /> {t('form.default_instance', { kind })}
	</label>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
	<div class="flex gap-2">
		<Button type="submit" disabled={busy || !profileId || !rootFolder}>{busy ? t('common.working') : t('common.save')}</Button>
		<Button type="button" variant="outline" onclick={onCancel}>{t('common.cancel')}</Button>
	</div>
</form>

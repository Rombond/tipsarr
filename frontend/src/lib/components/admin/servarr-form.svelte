<script lang="ts">
	import { api, unwrap, type Schemas } from '$lib/api/client';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

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
	let probe = $state<Schemas['ProbeResult'] | null>(null);
	let status = $state<string | null>(null);
	let error = $state<string | null>(null);
	let busy = $state(false);

	async function test() {
		busy = true;
		error = status = null;
		try {
			probe = await unwrap(api.POST('/admin/servarr/probe', { body: { kind, url: url.trim(), apiKey: apiKey.trim(), id: instance?.id } }));
			status = `Connected to ${probe.appName} ${probe.version}. Pick a quality profile and root folder.`;
			if (!probe.profiles.some((p) => p.id === profileId)) profileId = probe.profiles[0]?.id ?? 0;
			if (!probe.rootFolders.some((f) => f.path === rootFolder)) rootFolder = probe.rootFolders[0]?.path ?? '';
		} catch (e) {
			probe = null;
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			const body = { kind, name: name.trim(), url: url.trim(), apiKey: apiKey.trim(), qualityProfileId: profileId, rootFolder, isDefault };
			if (instance) await unwrap(api.PUT('/admin/servarr/{id}', { params: { path: { id: instance.id } }, body }));
			else await unwrap(api.POST('/admin/servarr', { body }));
			onSaved();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			busy = false;
		}
	}

	const selectClass = 'h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm';
</script>

<form class="grid gap-3 rounded-lg border border-border p-3" onsubmit={save}>
	<Input placeholder="Name" bind:value={name} required />
	<Input placeholder="URL (http://host:{kind === 'radarr' ? 7878 : 8989})" type="url" bind:value={url} required />
	<Input
		placeholder={instance ? 'API key (leave empty to keep the saved one)' : 'API key'}
		bind:value={apiKey}
		autocomplete="off"
		required={!instance}
	/>
	<div class="flex items-center gap-2">
		<Button type="button" variant="outline" size="sm" disabled={busy || !url.trim() || (!apiKey.trim() && !instance)} onclick={test}>
			Test connection
		</Button>
		{#if status}<span class="text-xs text-muted-foreground">{status}</span>{/if}
	</div>

	{#if probe}
		<label class="grid gap-1 text-xs">
			Quality profile
			<select class={selectClass} bind:value={profileId}>
				{#each probe.profiles as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
			</select>
		</label>
		<label class="grid gap-1 text-xs">
			Root folder
			<select class={selectClass} bind:value={rootFolder}>
				{#each probe.rootFolders as f (f.id)}<option value={f.path}>{f.path}</option>{/each}
			</select>
		</label>
	{:else if instance}
		<p class="text-xs text-muted-foreground">Profile #{instance.qualityProfileId} · {instance.rootFolder}. Test the connection to change them.</p>
	{/if}

	<label class="flex items-center gap-2 text-sm">
		<input type="checkbox" bind:checked={isDefault} /> Default {kind} instance
	</label>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
	<div class="flex gap-2">
		<Button type="submit" disabled={busy || !profileId || !rootFolder}>{busy ? 'Working…' : 'Save'}</Button>
		<Button type="button" variant="outline" onclick={onCancel}>Cancel</Button>
	</div>
</form>

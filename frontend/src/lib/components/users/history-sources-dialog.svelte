<script lang="ts">
	import { untrack } from 'svelte';
	// Whose watch history a person's recommendations also use (a shared TV account, say). Admin only.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';

	const MAX = 5;

	let {
		open = $bindable(false),
		user,
		users,
	}: {
		open?: boolean;
		user: Schemas['User'];
		users: Schemas['User'][];
	} = $props();

	let chosen = $state<string[]>([]);
	let loaded = $state(false); // the sources shown are the ones of this person, read from the server
	let loading = $state(false);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let loadedFor = '';

	const others = $derived(users.filter((u) => u.id !== user.id));
	const names = $derived(chosen.map((id) => users.find((u) => u.id === id)?.name ?? id).join(', '));

	async function load(id: string) {
		loading = true;
		loaded = false;
		chosen = []; // never keep what another person's dialog showed
		error = null;
		try {
			const r = await unwrap(api.GET('/admin/users/{id}/history-sources', { params: { path: { id } } }));
			if (id === user.id) {
				chosen = r.sources.map((s) => s.id);
				loaded = true;
			}
		} catch (e) {
			error = errorText(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (!open) {
			loadedFor = '';
			return;
		}
		if (loadedFor === user.id) return;
		loadedFor = user.id;
		const id = user.id;
		untrack(() => load(id));
	});

	function toggle(id: string, on: boolean) {
		chosen = on ? [...chosen, id] : chosen.filter((x) => x !== id);
	}

	async function save() {
		saving = true;
		error = null;
		try {
			await unwrap(api.PUT('/admin/users/{id}/history-sources', { params: { path: { id: user.id } }, body: { sources: chosen } }));
			toast.success(t('common.saved'));
			open = false;
		} catch (e) {
			error = errorText(e);
		} finally {
			saving = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{t('users.sources_title', { name: user.name })}</Dialog.Title>
			<Dialog.Description>{t('users.sources_hint', { name: user.name })}</Dialog.Description>
		</Dialog.Header>
		<div class="flex gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm" role="note">
			<p>{chosen.length ? t('users.sources_warning', { name: user.name, sources: names }) : t('users.sources_warning_none', { name: user.name })}</p>
		</div>
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<div class="grid gap-1.5" aria-busy={loading}>
			{#each others as u (u.id)}
				{@const on = chosen.includes(u.id)}
				<label class="flex cursor-pointer items-center gap-2 rounded-md border border-border p-2 text-sm {!on && chosen.length >= MAX ? 'cursor-not-allowed opacity-60' : ''}">
					<input type="checkbox" checked={on} disabled={loading || !loaded || (!on && chosen.length >= MAX)} onchange={(e) => toggle(u.id, e.currentTarget.checked)} />
					{u.name}
				</label>
			{:else}
				<p class="text-sm text-muted-foreground">{t('users.sources_none')}</p>
			{/each}
		</div>
		<p class="text-xs text-muted-foreground">{t('users.sources_import_hint', { max: MAX })}</p>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button disabled={saving || loading || !loaded} onclick={save}>{saving ? t('common.saving') : t('common.save')}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

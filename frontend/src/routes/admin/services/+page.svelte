<script lang="ts">
	import { t, type Key } from '$lib/i18n/index.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import ServarrForm from '$lib/components/admin/servarr-form.svelte';

	let instances = $state<Schemas['InstanceView'][]>([]);
	let error = $state<string | null>(null);
	let editing = $state<{ kind: 'radarr' | 'sonarr'; instance: Schemas['InstanceView'] | null } | null>(null);

	async function load() {
		try {
			instances = await unwrap(api.GET('/admin/servarr'));
		} catch (e) {
			error = errorText(e);
		}
	}

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		load();
	});

	async function remove(i: Schemas['InstanceView']) {
		if (!confirm(t('services.remove_confirm', { name: i.name }))) return;
		try {
			await expectOk(api.DELETE('/admin/servarr/{id}', { params: { path: { id: i.id } } }));
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}

	const kinds = [
		{ kind: 'radarr', title: 'services.radarr' },
		{ kind: 'sonarr', title: 'services.sonarr' },
	] as const satisfies { kind: string; title: Key }[];
</script>

<svelte:head>
	<title>{t('nav.services')} · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] items-start gap-4 lg:grid-cols-2">
	<h1 class="font-bold text-3xl lg:col-span-2">{t('services.title')}</h1>
	{#if auth.dryRun}
		<p class="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm lg:col-span-2">
			{t('services.dry_notice')}
		</p>
	{/if}
	{#if error}<p class="text-sm text-destructive lg:col-span-2">{error}</p>{/if}

	{#each kinds as k (k.kind)}
		<Card>
			<CardHeader>
				<CardTitle>{t(k.title)}</CardTitle>
				<CardDescription>{t('services.default_desc')}</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-3">
				{#each instances.filter((i) => i.kind === k.kind) as i (i.id)}
					{#if editing?.instance?.id === i.id}
						<ServarrForm kind={k.kind} instance={i} onSaved={() => { editing = null; load(); }} onCancel={() => (editing = null)} />
					{:else}
						<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3 text-sm">
							<div class="grid gap-0.5">
								<span class="flex items-center gap-2 font-medium">{i.name}{#if i.isDefault}<Badge variant="secondary">{t('services.default')}</Badge>{/if}</span>
								<span class="font-mono text-xs text-muted-foreground">{i.url}</span>
								<span class="text-xs text-muted-foreground">{t('services.profile_line', { id: i.qualityProfileId, folder: i.rootFolder })}</span>
							</div>
							<div class="flex gap-2">
								<Button size="sm" variant="outline" onclick={() => (editing = { kind: k.kind, instance: i })}>{t('common.edit')}</Button>
								<Button size="sm" variant="ghost" onclick={() => remove(i)}>{t('common.remove')}</Button>
							</div>
						</div>
					{/if}
				{/each}
				{#if editing && editing.instance === null && editing.kind === k.kind}
					<ServarrForm kind={k.kind} onSaved={() => { editing = null; load(); }} onCancel={() => (editing = null)} />
				{:else}
					<div>
						<Button size="sm" variant="outline" onclick={() => (editing = { kind: k.kind, instance: null })}>{t(k.kind === 'radarr' ? 'services.add_radarr' : 'services.add_sonarr')}</Button>
					</div>
				{/if}
			</CardContent>
		</Card>
	{/each}
</div>

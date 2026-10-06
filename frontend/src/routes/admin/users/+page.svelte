<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { LOCALES } from '$lib/i18n/index.svelte';
	import { fmtDate } from '$lib/i18n/format';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	let users = $state<Schemas['User'][]>([]);
	let error = $state<string | null>(null);
	let message = $state<string | null>(null);
	let busy = $state(false);

	async function load() {
		try {
			users = await unwrap(api.GET('/admin/users'));
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

	async function patch(u: Schemas['User'], body: { role?: 'admin' | 'user'; region?: string; language?: string }) {
		error = message = null;
		try {
			const updated = await unwrap(api.PATCH('/admin/users/{id}', { params: { path: { id: u.id } }, body }));
			users = users.map((x) => (x.id === u.id ? updated : x));
		} catch (e) {
			error = errorText(e);
			await load(); // restore what the server has
		}
	}

	async function importUsers() {
		busy = true;
		error = message = null;
		try {
			const r = await unwrap(api.POST('/admin/users/import'));
			message = t('users.imported', { created: r.created, total: r.total });
			await load();
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	const when = (unix: number) => (unix ? fmtDate(unix) : t('common.never'));
</script>

<svelte:head>
	<title>{t('users.title')} · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h1 class="font-bold text-3xl">{t('users.title')}</h1>
		<Button variant="outline" size="sm" disabled={busy} onclick={importUsers}>{t('users.import')}</Button>
	</div>
	<p class="text-sm text-muted-foreground">
		{t('users.intro')}
	</p>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
	{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}

	<div class="grid gap-2">
		{#each users as u (u.id)}
			<div class="flex flex-wrap items-center gap-3 rounded-lg border border-border p-3 text-sm">
				<div class="min-w-32 flex-1">
					<div class="font-medium">{u.name}{#if u.id === auth.user?.id} <span class="text-xs text-muted-foreground">{t('common.you')}</span>{/if}</div>
					<div class="text-xs text-muted-foreground">{t('users.last_signin', { when: when(u.lastLoginAt) })}</div>
				</div>
				<SimpleSelect
					label={t('users.role')}
					value={u.role}
					disabled={u.id === auth.user?.id}
					options={[{ value: 'user', label: t('users.role_user') }, { value: 'admin', label: t('users.role_admin') }]}
					onchange={(v) => patch(u, { role: v as 'admin' | 'user' })}
					class="h-8 min-w-24"
				/>
				<Input
					class="h-8 w-20"
					placeholder={t('users.region')}
					maxlength={2}
					value={u.region}
					aria-label={t('users.region')}
					onchange={(e) => patch(u, { region: e.currentTarget.value.trim().toUpperCase() })}
				/>
				<SimpleSelect
					label={t('lang.title')}
					value={u.language}
					options={[{ value: '', label: t('lang.auto') }, ...LOCALES.map((l) => ({ value: l.tag, label: l.name }))]}
					onchange={(v) => patch(u, { language: v })}
					class="h-8 min-w-32"
				/>
			</div>
		{/each}
	</div>
</div>

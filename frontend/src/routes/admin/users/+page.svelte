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
	import UserAvatar from '$lib/components/users/user-avatar.svelte';
	import UserAccessDialog from '$lib/components/users/user-access-dialog.svelte';
	import TeaserHistoryDialog from '$lib/components/users/teaser-history-dialog.svelte';
	import HistorySourcesDialog from '$lib/components/users/history-sources-dialog.svelte';
	import PermissionDefaults from '$lib/components/users/permission-defaults.svelte';
	import type { Flag } from '$lib/permissions';

	let users = $state<Schemas['User'][]>([]);
	let error = $state<string | null>(null);
	let message = $state<string | null>(null);
	let busy = $state(false);
	let flags = $state<Flag[]>([]);
	let defaults = $state<Schemas['PermissionDefaults'] | null>(null);
	let editing = $state<Schemas['User'] | null>(null);
	let editOpen = $state(false);
	let historyUser = $state<Schemas['User'] | null>(null);
	let historyOpen = $state(false);
	let sourcesUser = $state<Schemas['User'] | null>(null);
	let sourcesOpen = $state(false);

	async function load() {
		try {
			users = await unwrap(api.GET('/admin/users'));
		} catch (e) {
			error = errorText(e);
		}
	}

	onMount(() => {
		if (!auth.canManageUsers) {
			goto('/discover');
			return;
		}
		load();
		unwrap(api.GET('/admin/permissions')).then((p) => {
			flags = p.flags as Flag[];
			defaults = p.defaults;
		}).catch((e) => (error = errorText(e)));
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
				<a href="/users/{u.id}" class="shrink-0" tabindex="-1" aria-hidden="true"><UserAvatar id={u.id} name={u.name} class="size-10" /></a>
				<div class="min-w-32 flex-1">
					<div class="font-medium"><a href="/users/{u.id}" class="hover:underline">{u.name}</a>{#if u.id === auth.user?.id} <span class="text-xs text-muted-foreground">{t('common.you')}</span>{/if}</div>
					<div class="text-xs text-muted-foreground">{t('users.last_signin', { when: when(u.lastLoginAt) })}</div>
				</div>
				<SimpleSelect
					label={t('users.role')}
					value={u.role}
					disabled={u.id === auth.user?.id || !auth.isAdmin}
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
				<Button variant="outline" size="sm" disabled={!flags.length} onclick={() => { editing = u; editOpen = true; }}>{t('users.permissions')}</Button>
				<Button variant="outline" size="sm" onclick={() => { historyUser = u; historyOpen = true; }}>{t('users.teasers_history')}</Button>
				{#if auth.isAdmin}<Button variant="outline" size="sm" onclick={() => { sourcesUser = u; sourcesOpen = true; }}>{t('users.sources')}</Button>{/if}
			</div>
		{/each}
	</div>

	{#if defaults && flags.length}
		<PermissionDefaults {flags} bind:defaults />
	{/if}
</div>

{#if editing && flags.length}
	<UserAccessDialog
		bind:open={editOpen}
		user={editing}
		{flags}
		{defaults}
		onsaved={(updated) => (users = users.map((x) => (x.id === updated.id ? updated : x)))}
	/>
{/if}

{#if historyUser}
	<TeaserHistoryDialog bind:open={historyOpen} user={historyUser} />
{/if}

{#if sourcesUser}
	<HistorySourcesDialog bind:open={sourcesOpen} user={sourcesUser} {users} />
{/if}

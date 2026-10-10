<script lang="ts">
	// Permissions and request limits of one person.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import PermissionEditor from './permission-editor.svelte';
	import QuotaFields, { type QuotaValues } from './quota-fields.svelte';
	import type { Flag } from '$lib/permissions';

	let {
		open = $bindable(false),
		user,
		flags,
		defaults,
		onsaved,
	}: {
		open?: boolean;
		user: Schemas['User'];
		flags: Flag[];
		defaults: Schemas['PermissionDefaults'] | null;
		onsaved: (u: Schemas['User']) => void;
	} = $props();

	let mask = $state(0);
	let quota = $state<QuotaValues>({ movieLimit: '', movieDays: '', tvLimit: '', tvDays: '' });
	let busy = $state(false);
	let error = $state<string | null>(null);

	const text = (n?: number | null) => (n === undefined || n === null ? '' : String(n));
	const num = (s: string | number | null | undefined) => (s === '' || s === null || s === undefined ? undefined : Number(s));

	$effect(() => {
		if (!open) return;
		mask = user.permissions;
		quota = { movieLimit: text(user.movieQuotaLimit), movieDays: text(user.movieQuotaDays), tvLimit: text(user.tvQuotaLimit), tvDays: text(user.tvQuotaDays) };
		error = null;
	});

	const isAdminUser = $derived(user.role === 'admin');
	const isSelf = $derived(user.id === auth.user?.id);
	const readOnly = $derived(isAdminUser || isSelf);

	async function save() {
		busy = true;
		error = null;
		try {
			const updated = await unwrap(
				api.PATCH('/admin/users/{id}', {
					params: { path: { id: user.id } },
					body: {
						permissions: mask,
						quota: { movieLimit: num(quota.movieLimit), movieDays: num(quota.movieDays), tvLimit: num(quota.tvLimit), tvDays: num(quota.tvDays) },
					},
				}),
			);
			open = false;
			toast.success(t('users.permissions_saved'));
			onsaved(updated);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-xl">
		<Dialog.Header>
			<Dialog.Title>{t('users.permissions_title', { name: user.name })}</Dialog.Title>
			{#if isAdminUser}<Dialog.Description>{t('users.permissions_admin')}</Dialog.Description>
			{:else if isSelf}<Dialog.Description>{t('users.permissions_own')}</Dialog.Description>
			{:else if !auth.isAdmin}<Dialog.Description>{t('users.permissions_keys')}</Dialog.Description>{/if}
		</Dialog.Header>
		<div class="grid gap-5">
			<PermissionEditor {flags} bind:mask disabled={readOnly} keysLocked={!auth.isAdmin} />
			<div class="grid gap-2">
				<h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t('users.quota_title')}</h3>
				<QuotaFields bind:values={quota} defaults={defaults ? { movieLimit: defaults.movieQuotaLimit, movieDays: defaults.movieQuotaDays, tvLimit: defaults.tvQuotaLimit, tvDays: defaults.tvQuotaDays } : null} disabled={readOnly} />
			</div>
		</div>
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button disabled={busy || readOnly} onclick={save}>{busy ? t('common.saving') : t('common.save')}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

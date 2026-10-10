<script lang="ts">
	// What people get when they sign in for the first time. Only admins can change it.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import PermissionEditor from './permission-editor.svelte';
	import QuotaFields, { type QuotaValues } from './quota-fields.svelte';
	import type { Flag } from '$lib/permissions';

	let { flags, defaults = $bindable() }: { flags: Flag[]; defaults: Schemas['PermissionDefaults'] } = $props();

	let mask = $state(0);
	let quota = $state<QuotaValues>({ movieLimit: '', movieDays: '', tvLimit: '', tvDays: '' });
	let busy = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		mask = defaults.permissions;
		quota = { movieLimit: String(defaults.movieQuotaLimit), movieDays: String(defaults.movieQuotaDays), tvLimit: String(defaults.tvQuotaLimit), tvDays: String(defaults.tvQuotaDays) };
	});

	const n = (s: string) => (s === '' ? 0 : Number(s));

	async function save() {
		busy = true;
		error = null;
		try {
			defaults = await unwrap(
				api.PUT('/admin/permissions/defaults', {
					body: { permissions: mask, movieQuotaLimit: n(quota.movieLimit), movieQuotaDays: n(quota.movieDays), tvQuotaLimit: n(quota.tvLimit), tvQuotaDays: n(quota.tvDays) },
				}),
			);
			toast.success(t('common.saved'));
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>{t('users.defaults_title')}</CardTitle>
		<CardDescription>{t('users.defaults_desc')}</CardDescription>
	</CardHeader>
	<CardContent class="grid gap-5">
		<PermissionEditor {flags} bind:mask disabled={!auth.isAdmin} />
		<div class="grid gap-2">
			<h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t('users.quota_title')}</h3>
			<QuotaFields bind:values={quota} disabled={!auth.isAdmin} />
		</div>
		{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
		{#if auth.isAdmin}
			<div><Button disabled={busy} onclick={save}>{busy ? t('common.saving') : t('common.save')}</Button></div>
		{/if}
	</CardContent>
</Card>

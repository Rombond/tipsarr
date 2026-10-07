<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { auth } from '$lib/stores/auth.svelte';

	const settings = $derived(admin.settings);

	async function toggleFolderChoice(on: boolean) {
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { userFolderChoice: on } }));
			auth.userFolderChoice = on;
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			admin.settings = await unwrap(api.GET('/admin/settings'));
		}
	}

	async function toggleImport(on: boolean) {
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { servarrAutoImport: on } }));
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			admin.settings = await unwrap(api.GET('/admin/settings'));
		}
	}
</script>

{#if settings}
	<Card>
		<CardHeader>
			<CardTitle>{t('settings.useropts_title')}</CardTitle>
			<CardDescription>{t('settings.useropts_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<label class="flex cursor-pointer items-center gap-2 text-sm">
				<input type="checkbox" checked={settings.userFolderChoice} onchange={(e) => toggleFolderChoice(e.currentTarget.checked)} />
				{t('settings.useropts_toggle')}
			</label>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.import_title')}</CardTitle>
			<CardDescription>{t('settings.import_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<label class="flex cursor-pointer items-center gap-2 text-sm">
				<input type="checkbox" checked={settings.servarrAutoImport} onchange={(e) => toggleImport(e.currentTarget.checked)} />
				{t('settings.import_toggle')}
			</label>
		</CardContent>
	</Card>
{:else}
	<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
{/if}

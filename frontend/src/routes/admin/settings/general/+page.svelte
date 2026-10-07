<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { i18n, LOCALES } from '$lib/i18n/index.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	const settings = $derived(admin.settings);

	// app-wide default language
	let defaultLang = $state('');
	let regions = $state('');
	$effect(() => {
		if (!settings) return;
		defaultLang = settings.defaultLanguage;
		regions = settings.boxofficeRegions;
	});

	async function saveDefaultLanguage(v: string) {
		defaultLang = v;
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { defaultLanguage: v } }));
			i18n.setAppDefault(v);
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			admin.settings = await unwrap(api.GET('/admin/settings'));
		}
	}

	async function saveRegions(e: SubmitEvent) {
		e.preventDefault();
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { boxofficeRegions: regions } }));
			regions = admin.settings.boxofficeRegions;
			toast.success(t('settings.regions_saved'));
		} catch (err) {
			toast.error(errorText(err));
		}
	}
</script>

{#if settings}
	<Card>
		<CardHeader>
			<CardTitle>{t('settings.lang_title')}</CardTitle>
			<CardDescription>{t('settings.lang_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<SimpleSelect
				label={t('settings.lang_title')}
				value={defaultLang}
				options={[{ value: '', label: t('settings.lang_browser') }, ...LOCALES.map((l) => ({ value: l.tag, label: l.name }))]}
				onchange={saveDefaultLanguage}
				class="w-full max-w-xs"
			/>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.regions_title')}</CardTitle>
			<CardDescription>
				{t('settings.regions_desc')}
			</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="flex gap-2" onsubmit={saveRegions}>
				<Input bind:value={regions} placeholder="US" autocomplete="off" />
				<Button type="submit" variant="outline">{t('common.save')}</Button>
			</form>
		</CardContent>
	</Card>
{:else}
	<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
{/if}

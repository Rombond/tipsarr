<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import EyeIcon from '@lucide/svelte/icons/eye';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import RefreshIcon from '@lucide/svelte/icons/refresh-cw';

	const settings = $derived(admin.settings);
	// the settings only say whether a key exists; the key itself is read on demand (admin role only)
	const on = $derived(!!settings?.apiKeyConfigured);
	let key = $state('');
	let shown = $state(false);
	let busy = $state(false);

	async function secret(action: 'reveal' | 'regenerate'): Promise<string> {
		const params = { path: { name: 'api_key' as const } };
		const res = await unwrap(action === 'reveal' ? api.POST('/admin/secrets/{name}/reveal', { params }) : api.POST('/admin/secrets/{name}/regenerate', { params }));
		return res.value;
	}

	async function reveal(): Promise<string> {
		if (!key) key = await secret('reveal');
		return key;
	}

	async function toggleShown() {
		try {
			if (!shown) await reveal();
			shown = !shown;
		} catch (e) {
			toast.error(errorText(e));
		}
	}

	async function put(body: Schemas['UpdateSettingsInputBody']) {
		busy = true;
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body }));
			toast.success(t('common.saved'));
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			busy = false;
		}
	}

	async function toggle(enable: boolean) {
		key = '';
		shown = false;
		await put({ apiKeyEnabled: enable });
	}

	async function regenerate() {
		if (!confirm(t('apikey.regenerate_confirm'))) return;
		busy = true;
		try {
			key = await secret('regenerate');
			shown = true; // show the new key once
			admin.settings = await unwrap(api.GET('/admin/settings'));
			toast.success(t('common.saved'));
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			busy = false;
		}
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(await reveal());
			toast.success(t('apikey.copied'));
		} catch {
			shown = true; // no clipboard (plain http): show it so it can be selected by hand
		}
	}
</script>

{#if settings}
	<Card>
		<CardHeader>
			<CardTitle>{t('apikey.title')}</CardTitle>
			<CardDescription>{t('apikey.desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-4">
			<label class="flex cursor-pointer items-center gap-2 text-sm">
				<input type="checkbox" checked={on} disabled={busy} onchange={(e) => toggle(e.currentTarget.checked)} />
				{t('apikey.toggle')}
			</label>

			{#if on}
				<div class="flex gap-2">
					<Input value={shown ? key : '••••••••••••••••'} readonly type={shown ? 'text' : 'password'} autocomplete="off" class="font-mono" aria-label={t('apikey.title')} />
					<Button variant="outline" size="icon" onclick={toggleShown} aria-label={shown ? t('apikey.hide') : t('apikey.show')} title={shown ? t('apikey.hide') : t('apikey.show')}>
						{#if shown}<EyeOffIcon />{:else}<EyeIcon />{/if}
					</Button>
					<Button variant="outline" size="icon" onclick={copy} aria-label={t('apikey.copy')} title={t('apikey.copy')}>
						<CopyIcon />
					</Button>
					<Button variant="outline" size="icon" onclick={regenerate} disabled={busy} aria-label={t('apikey.regenerate')} title={t('apikey.regenerate')}>
						<RefreshIcon />
					</Button>
				</div>
				<p class="text-xs text-muted-foreground">{t('apikey.usage')}</p>
			{/if}
		</CardContent>
	</Card>
{/if}

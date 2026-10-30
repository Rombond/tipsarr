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
	const key = $derived(settings?.apiKey ?? '');
	let shown = $state(false);
	let busy = $state(false);

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

	const toggle = (on: boolean) => put({ apiKeyEnabled: on });

	function regenerate() {
		if (confirm(t('apikey.regenerate_confirm'))) put({ regenerateApiKey: true });
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(key);
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
				<input type="checkbox" checked={!!key} disabled={busy} onchange={(e) => toggle(e.currentTarget.checked)} />
				{t('apikey.toggle')}
			</label>

			{#if key}
				<div class="flex gap-2">
					<Input value={key} readonly type={shown ? 'text' : 'password'} autocomplete="off" class="font-mono" aria-label={t('apikey.title')} />
					<Button variant="outline" size="icon" onclick={() => (shown = !shown)} aria-label={shown ? t('apikey.hide') : t('apikey.show')} title={shown ? t('apikey.hide') : t('apikey.show')}>
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

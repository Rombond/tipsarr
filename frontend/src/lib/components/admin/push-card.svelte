<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { fmtDateTime } from '$lib/i18n/format';

	// Mobile push through the relay. The store apps only get pushes from the official relay, so the relay address and a
	// pasted key sit under "Advanced" (a self-built app or your own relay).
	const settings = $derived(admin.settings);
	const relayInUse = $derived(settings ? settings.pushRelayUrl || settings.pushDefaultRelayUrl : '');
	// a custom relay or a hand-made key: show the advanced block open
	const custom = $derived(
		!!settings &&
			((!!settings.pushRelayUrl && settings.pushRelayUrl !== settings.pushDefaultRelayUrl) || (settings.pushRelayKeyConfigured && !settings.pushKeyPublic)),
	);

	// editable copy of the saved address (reset whenever the settings change)
	let pushUrl = $derived(settings?.pushRelayUrl ?? '');
	let pushKey = $state('');
	let registering = $state(false);
	let registerError = $state<string | null>(null);

	onMount(() => {
		admin.loadSettings();
	});

	async function savePush(e: SubmitEvent) {
		e.preventDefault();
		if (!settings) return;
		try {
			admin.settings = await unwrap(
				api.PUT('/admin/settings', {
					// an empty field means the official relay (an empty value would clear the address and turn push off)
					body: { pushRelayUrl: pushUrl.trim() || settings.pushDefaultRelayUrl, ...(pushKey.trim() ? { pushRelayKey: pushKey.trim() } : {}) },
				}),
			);
			pushKey = '';
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		}
	}

	async function registerPush() {
		if (!settings) return;
		registering = true;
		registerError = null;
		try {
			// empty = the official relay
			admin.settings = await unwrap(api.POST('/admin/push/register', { body: { relayUrl: settings.pushRelayUrl } }));
			toast.success(t('settings.push_registered'));
		} catch (err) {
			registerError = errorText(err);
		} finally {
			registering = false;
		}
	}

	async function togglePush(on: boolean, key: 'pushEnabled' | 'pushText' = 'pushEnabled') {
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { [key]: on } }));
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			admin.settings = await unwrap(api.GET('/admin/settings'));
		}
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>{t('settings.push_title')}</CardTitle>
		<CardDescription>{t('settings.push_desc')}</CardDescription>
	</CardHeader>
	<CardContent class="grid gap-3">
		{#if !settings}
			<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
		{:else}
			{#if settings.pushPausedUntil > 0}
				<p class="rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-sm" role="status">
					{t('settings.push_paused', { time: fmtDateTime(settings.pushPausedUntil) })}
				</p>
			{/if}
			<p class="text-sm text-muted-foreground">{t('settings.push_relay_in_use', { url: relayInUse })}</p>
			<div class="grid gap-1">
				<Button type="button" class="w-fit" disabled={registering} onclick={registerPush}>{t('settings.push_register')}</Button>
				<p class="text-xs text-muted-foreground">{t('settings.push_register_hint')}</p>
				{#if registerError}<p class="text-sm text-destructive" role="alert">{registerError}</p>{/if}
			</div>
			{#if settings.pushKeyPublic}
				<Badge variant="secondary" class="w-fit">
					{settings.pushDailyQuota > 0 ? t('settings.push_key_public', { count: settings.pushDailyQuota }) : t('settings.push_key_public_unlimited')}
				</Badge>
			{/if}
			<label class="flex cursor-pointer items-center gap-2 text-sm">
				<input type="checkbox" checked={settings.pushEnabled} onchange={(e) => togglePush(e.currentTarget.checked)} />
				{t('settings.push_toggle')}
			</label>
			<label class="flex items-center gap-2 text-sm {settings.pushKeyPublic ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}">
				<input
					type="checkbox"
					checked={settings.pushText && !settings.pushKeyPublic}
					disabled={settings.pushKeyPublic}
					onchange={(e) => togglePush(e.currentTarget.checked, 'pushText')}
				/>
				{t('settings.push_text')}
			</label>
			<p class="text-xs text-muted-foreground">
				{settings.pushKeyPublic ? t('settings.push_text_public') : t('settings.push_text_hint')}
			</p>

			<details class="rounded-md border border-border text-sm" open={custom}>
				<summary class="cursor-pointer px-3 py-2 font-medium">{t('settings.push_advanced')}</summary>
				<form class="grid gap-2 border-t border-border p-3" onsubmit={savePush}>
					<p class="text-xs text-muted-foreground">{t('settings.push_advanced_hint')}</p>
					<Input bind:value={pushUrl} placeholder={settings.pushDefaultRelayUrl} type="url" autocomplete="off" aria-label={t('settings.push_relay_url')} />
					<Input
						bind:value={pushKey}
						placeholder={settings.pushRelayKeyConfigured ? t('settings.push_key_saved') : t('settings.push_key_placeholder')}
						type="password"
						autocomplete="off"
						aria-label={t('settings.push_key_placeholder')}
					/>
					<Button type="submit" variant="outline" class="w-fit">{t('common.save')}</Button>
				</form>
			</details>
		{/if}
	</CardContent>
</Card>

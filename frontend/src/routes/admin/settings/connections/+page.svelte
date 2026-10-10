<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';

	const settings = $derived(admin.settings);
	let tmdbKey = $state('');
	let jellyfinKey = $state('');
	let publicUrl = $state('');
	let message = $state<string | null>(null);
	let saving = $state(false);
	let pushUrl = $state('');
	let pushKey = $state('');

	$effect(() => {
		if (settings) {
			publicUrl = settings.jellyfinPublicUrl;
			pushUrl = settings.pushRelayUrl;
		}
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		admin.error = message = null;
		try {
			admin.settings = await unwrap(
				api.PUT('/admin/settings', {
					body: {
						...(tmdbKey.trim() ? { tmdbApiKey: tmdbKey.trim() } : {}),
						...(jellyfinKey.trim() ? { jellyfinApiKey: jellyfinKey.trim() } : {}),
					},
				}),
			);
			tmdbKey = jellyfinKey = '';
			message = t('profile.saved');
			await admin.refreshSync();
		} catch (err) {
			admin.error = errorText(err);
		} finally {
			saving = false;
		}
	}

	async function savePublicUrl(e: SubmitEvent) {
		e.preventDefault();
		admin.error = null;
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { jellyfinPublicUrl: publicUrl.trim() } }));
			publicUrl = admin.settings.jellyfinPublicUrl;
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		}
	}

	async function savePush(e: SubmitEvent) {
		e.preventDefault();
		admin.error = null;
		try {
			admin.settings = await unwrap(
				api.PUT('/admin/settings', {
					body: { pushRelayUrl: pushUrl.trim(), ...(pushKey.trim() ? { pushRelayKey: pushKey.trim() } : {}) },
				}),
			);
			pushKey = '';
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		}
	}

	async function togglePush(on: boolean) {
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body: { pushEnabled: on } }));
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
			<CardTitle>{t('settings.status')}</CardTitle>
		</CardHeader>
		<CardContent class="grid gap-1.5 text-sm">
			<div class="flex justify-between border-b border-border py-1.5">
				<span class="text-muted-foreground">{t('settings.jellyfin')}</span>
				<span class="font-mono">{settings.jellyfinUrl}</span>
			</div>
			<div class="flex items-center justify-between py-1.5">
				<span class="text-muted-foreground">{t('settings.dry_run')}</span>
				<Badge variant={settings.dryRun ? 'default' : 'destructive'}>{settings.dryRun ? t('settings.dry_run_on') : t('settings.dry_run_off')}</Badge>
			</div>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.keys_title')}</CardTitle>
			<CardDescription>
				{t('settings.keys_desc', {
					tmdb: settings.tmdbConfigured ? t('settings.tmdb_saved') : t('settings.tmdb_missing'),
					jellyfin: settings.jellyfinApiKeyConfigured ? t('settings.jf_saved') : t('settings.jf_missing'),
				})}
			</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="grid gap-3" onsubmit={save}>
				<Input placeholder={t('settings.tmdb_placeholder')} bind:value={tmdbKey} autocomplete="off" />
				<Input placeholder={t('settings.jf_placeholder')} bind:value={jellyfinKey} autocomplete="off" />
				{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}
				<Button type="submit" disabled={saving || (!tmdbKey.trim() && !jellyfinKey.trim())}>
					{saving ? t('common.saving') : t('common.save')}
				</Button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.links_title')}</CardTitle>
			<CardDescription>
				{t('settings.links_desc', { url: settings.jellyfinUrl })}
			</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="flex gap-2" onsubmit={savePublicUrl}>
				<Input bind:value={publicUrl} placeholder="https://jellyfin.example.org" type="url" autocomplete="off" />
				<Button type="submit" variant="outline">{t('common.save')}</Button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.push_title')}</CardTitle>
			<CardDescription>{t('settings.push_desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-3">
			<form class="grid gap-2" onsubmit={savePush}>
				<Input bind:value={pushUrl} placeholder="https://push.example.org" type="url" autocomplete="off" />
				<Input
					bind:value={pushKey}
					placeholder={settings.pushRelayKeyConfigured ? t('settings.push_key_saved') : t('settings.push_key_placeholder')}
					type="password"
					autocomplete="off"
				/>
				<Button type="submit" variant="outline">{t('common.save')}</Button>
			</form>
			<label class="flex cursor-pointer items-center gap-2 text-sm">
				<input type="checkbox" checked={settings.pushEnabled} onchange={(e) => togglePush(e.currentTarget.checked)} />
				{t('settings.push_toggle')}
			</label>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.services_title')}</CardTitle>
			<CardDescription>{t('settings.services_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<Button variant="outline" href="/admin/services">{t('settings.services_open')}</Button>
		</CardContent>
	</Card>
{:else}
	<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
{/if}

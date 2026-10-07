<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import DownloadIcon from '@lucide/svelte/icons/download';
	import { auth } from '$lib/stores/auth.svelte';

	const settings = $derived(admin.settings);

	// single sign-on (OpenID Connect)
	let ssoIssuer = $state('');
	let ssoClientId = $state('');
	let ssoSecret = $state('');
	let ssoGroup = $state('');
	let ssoClaim = $state('');
	let ssoBusy = $state(false);
	const callbackUrl = $derived(`${location.origin}/api/v1/auth/oidc/callback`);

	$effect(() => {
		if (!settings) return;
		ssoIssuer = settings.oidcIssuer;
		ssoClientId = settings.oidcClientId;
		ssoGroup = settings.oidcAdminGroup;
		ssoClaim = settings.oidcGroupsClaim;
	});

	async function saveSso(e: SubmitEvent) {
		e.preventDefault();
		ssoBusy = true;
		try {
			admin.settings = await unwrap(
				api.PUT('/admin/settings', {
					body: {
						oidcIssuer: ssoIssuer.trim(),
						oidcClientId: ssoClientId.trim(),
						oidcAdminGroup: ssoGroup.trim(),
						oidcGroupsClaim: ssoClaim.trim(),
						...(ssoSecret.trim() ? { oidcClientSecret: ssoSecret.trim() } : {}),
					},
				}),
			);
			ssoSecret = '';
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		} finally {
			ssoBusy = false;
		}
	}

	// LLDAP/LDAP profile pictures
	let ldapUrl = $state('');
	let ldapBindDn = $state('');
	let ldapBaseDn = $state('');
	let ldapPassword = $state('');
	let ldapBusy = $state(false);

	$effect(() => {
		if (!settings) return;
		ldapUrl = settings.ldapUrl;
		ldapBindDn = settings.ldapBindDn;
		ldapBaseDn = settings.ldapBaseDn;
	});

	async function saveLdap(e: SubmitEvent) {
		e.preventDefault();
		ldapBusy = true;
		try {
			admin.settings = await unwrap(
				api.PUT('/admin/settings', {
					body: {
						ldapUrl: ldapUrl.trim(),
						ldapBindDn: ldapBindDn.trim(),
						ldapBaseDn: ldapBaseDn.trim(),
						...(ldapPassword ? { ldapBindPassword: ldapPassword } : {}),
					},
				}),
			);
			ldapPassword = '';
			auth.avatarV++;
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		} finally {
			ldapBusy = false;
		}
	}

	async function importLdap() {
		ldapBusy = true;
		try {
			const r = await unwrap(api.POST('/admin/ldap/import-jellyfin'));
			admin.settings = await unwrap(api.GET('/admin/settings'));
			auth.avatarV++;
			if (r.connected) toast.success(t('settings.ldap_imported'));
			else toast.error(t('settings.ldap_imported_fail', { error: r.error ?? '' }));
		} catch (err) {
			toast.error(errorText(err));
		} finally {
			ldapBusy = false;
		}
	}
</script>

{#if settings}
	<Card>
		<CardHeader>
			<CardTitle>{t('settings.sso_title')}</CardTitle>
			<CardDescription>{t('settings.sso_desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-4">
			<div class="grid gap-1 text-xs">
				<span class="text-muted-foreground">{t('settings.sso_redirect')}</span>
				<code class="break-all rounded bg-muted p-2">{callbackUrl}</code>
			</div>
			<form class="grid gap-3 sm:grid-cols-2" onsubmit={saveSso}>
				<label class="grid gap-1 text-sm sm:col-span-2">{t('settings.sso_issuer')}<Input bind:value={ssoIssuer} placeholder="https://auth.example.org" type="url" autocomplete="off" /></label>
				<label class="grid gap-1 text-sm">{t('settings.sso_client_id')}<Input bind:value={ssoClientId} placeholder="tipsarr" autocomplete="off" /></label>
				<label class="grid gap-1 text-sm">{t('settings.sso_secret')}<Input bind:value={ssoSecret} type="password" autocomplete="off" placeholder={settings.oidcClientSecretConfigured ? t('settings.sso_secret_saved') : ''} /></label>
				<label class="grid gap-1 text-sm">{t('settings.sso_group')}<Input bind:value={ssoGroup} placeholder="tipsarr-admins" autocomplete="off" /></label>
				<label class="grid gap-1 text-sm">{t('settings.sso_claim')}<Input bind:value={ssoClaim} placeholder="groups" autocomplete="off" /></label>
				<div class="flex items-center gap-3 sm:col-span-2">
					<Button type="submit" disabled={ssoBusy}>{ssoBusy ? t('common.saving') : t('common.save')}</Button>
					<span class="text-xs text-muted-foreground">{t('settings.sso_fallback')} <code>/login?password=1</code></span>
				</div>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.ldap_title')}</CardTitle>
			<CardDescription>{t('settings.ldap_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="grid gap-3 sm:grid-cols-2" onsubmit={saveLdap}>
				<label class="grid gap-1 text-sm sm:col-span-2">{t('settings.ldap_url')}<Input bind:value={ldapUrl} placeholder="ldap://192.168.2.247:3890" autocomplete="off" /></label>
				<label class="grid gap-1 text-sm">{t('settings.ldap_bind')}<Input bind:value={ldapBindDn} placeholder="uid=svc_ldap,ou=people,dc=example,dc=com" autocomplete="off" /></label>
				<label class="grid gap-1 text-sm">{t('settings.ldap_password')}<Input bind:value={ldapPassword} type="password" autocomplete="off" placeholder={settings.ldapBindPasswordConfigured ? t('settings.sso_secret_saved') : ''} /></label>
				<label class="grid gap-1 text-sm sm:col-span-2">{t('settings.ldap_base')}<Input bind:value={ldapBaseDn} placeholder="ou=people,dc=example,dc=com" autocomplete="off" /></label>
				<div class="flex flex-wrap gap-2 sm:col-span-2">
					<Button type="submit" disabled={ldapBusy}>{ldapBusy ? t('common.saving') : t('common.save')}</Button>
					{#if settings.jellyfinApiKeyConfigured}
						<Button type="button" variant="outline" disabled={ldapBusy} onclick={importLdap}><DownloadIcon /> {t('settings.ldap_import')}</Button>
					{/if}
				</div>
			</form>
		</CardContent>
	</Card>
{:else}
	<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
{/if}

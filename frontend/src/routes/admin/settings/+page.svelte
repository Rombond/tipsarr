<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { hasKey } from '$lib/i18n/index.svelte';
	import { fmtDateTime } from '$lib/i18n/format';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';

	let settings = $state<Schemas['SettingsBody'] | null>(null);
	let sync = $state<Schemas['SyncStatusBody'] | null>(null);
	let tmdbKey = $state('');
	let jellyfinKey = $state('');
	let regions = $state('');
	let publicUrl = $state('');
	let message = $state<string | null>(null);
	let error = $state<string | null>(null);
	let saving = $state(false);

	const anyRunning = $derived(sync?.jobs.some((j) => j.running) ?? false);

	async function refreshSync() {
		try {
			sync = await unwrap(api.GET('/admin/sync'));
		} catch (e) {
			error = errorText(e);
		}
	}

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		unwrap(api.GET('/admin/settings'))
			.then((s) => {
				settings = s;
				regions = s.boxofficeRegions;
				publicUrl = s.jellyfinPublicUrl;
			})
			.catch((e) => (error = errorText(e)));
		refreshSync();
		const timer = setInterval(() => {
			if (anyRunning) refreshSync();
		}, 2000);
		return () => clearInterval(timer);
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = message = null;
		try {
			settings = await unwrap(
				api.PUT('/admin/settings', {
					body: {
						...(tmdbKey.trim() ? { tmdbApiKey: tmdbKey.trim() } : {}),
						...(jellyfinKey.trim() ? { jellyfinApiKey: jellyfinKey.trim() } : {}),
					},
				}),
			);
			tmdbKey = jellyfinKey = '';
			message = t('profile.saved');
			await refreshSync();
		} catch (err) {
			error = errorText(err);
		} finally {
			saving = false;
		}
	}

	async function saveRegions(e: SubmitEvent) {
		e.preventDefault();
		error = message = null;
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { boxofficeRegions: regions } }));
			regions = settings.boxofficeRegions;
			message = t('settings.regions_saved');
		} catch (err) {
			error = errorText(err);
		}
	}

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
			settings = await unwrap(
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
			settings = await unwrap(
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

	async function toggleUserOptions(on: boolean) {
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { userRequestOptions: on } }));
			auth.userRequestOptions = on;
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			settings = await unwrap(api.GET('/admin/settings'));
		}
	}

	async function toggleImport(on: boolean) {
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { servarrAutoImport: on } }));
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			settings = await unwrap(api.GET('/admin/settings'));
		}
	}

	async function savePublicUrl(e: SubmitEvent) {
		e.preventDefault();
		error = message = null;
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { jellyfinPublicUrl: publicUrl.trim() } }));
			publicUrl = settings.jellyfinPublicUrl;
			toast.success(t('common.saved'));
		} catch (err) {
			error = errorText(err);
			toast.error(error);
		}
	}

	type JobName = 'library-sync' | 'history-sync' | 'boxoffice-refresh' | 'servarr-import';

	async function runJob(job: JobName) {
		error = message = null;
		try {
			await unwrap(api.POST('/admin/sync/{job}', { params: { path: { job } } }));
			await refreshSync();
		} catch (err) {
			error = errorText(err);
		}
	}

	const when = (unix: number) => (unix ? fmtDateTime(unix) : t('common.never'));
	const statusLabel = (s: string) => (hasKey(`job.${s}`) ? t(`job.${s}` as 'job.ok') : s);
	const webhookUrl = $derived(settings ? `${location.origin}${settings.webhookPath}` : '');
</script>

<svelte:head>
	<title>{t('settings.title')} · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] items-start gap-4 lg:grid-cols-2">
	<h1 class="font-bold text-3xl lg:col-span-2">{t('settings.title')}</h1>

	{#if error}<p class="text-sm text-destructive lg:col-span-2">{error}</p>{/if}

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
	{/if}

	{#if settings}
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

		<Card class="lg:col-span-2">
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

		<Card class="lg:col-span-2">
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
					<div class="sm:col-span-2"><Button type="submit" disabled={ldapBusy}>{ldapBusy ? t('common.saving') : t('common.save')}</Button></div>
				</form>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>{t('settings.useropts_title')}</CardTitle>
				<CardDescription>{t('settings.useropts_desc')}</CardDescription>
			</CardHeader>
			<CardContent>
				<label class="flex cursor-pointer items-center gap-2 text-sm">
					<input type="checkbox" checked={settings.userRequestOptions} onchange={(e) => toggleUserOptions(e.currentTarget.checked)} />
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
	{/if}

	{#if sync}
		<Card>
			<CardHeader>
				<CardTitle>{t('settings.jobs_title')}</CardTitle>
				<CardDescription>
					{t('settings.jobs_desc', { movies: sync.movies, shows: sync.shows })}
					{#if !sync.canSync}{t('settings.jobs_need_key')}{/if}
				</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-3 text-sm">
				{#each sync.jobs as job (job.name)}
					<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3">
						<div class="grid gap-0.5">
							<span class="font-medium">{job.name}</span>
							<span class="text-xs text-muted-foreground">
								{job.running ? t('settings.job_running') : `${statusLabel(job.status)} · ${when(job.lastFinishedAt)}`}
								{#if job.message}· {job.message}{/if}
							</span>
							<span class="text-xs text-muted-foreground">{t('settings.job_every', { hours: Math.round(job.everySeconds / 3600) })}</span>
						</div>
						<Button
							variant="outline"
							size="sm"
							disabled={(job.name !== 'boxoffice-refresh' && job.name !== 'servarr-import' && !sync.canSync) || job.running}
							onclick={() => runJob(job.name as JobName)}
						>
							{t('settings.job_run')}
						</Button>
					</div>
				{/each}
				{#if anyRunning}<p class="text-xs text-muted-foreground">{t('settings.refreshing')}</p>{/if}
			</CardContent>
		</Card>

		{#if settings}
			<Card>
				<CardHeader>
					<CardTitle>{t('settings.hook_title')}</CardTitle>
					<CardDescription>
						{t('settings.hook_desc')}
					</CardDescription>
				</CardHeader>
				<CardContent class="grid gap-2 text-xs">
					<code class="break-all rounded bg-muted p-2">{webhookUrl}</code>
					<code class="break-all rounded bg-muted p-2">
						{'{"NotificationType":"{{NotificationType}}","UserId":"{{UserId}}","ItemType":"{{ItemType}}"}'}
					</code>
					<p class="text-muted-foreground">{t('settings.hook_secret')}</p>
				</CardContent>
			</Card>
		{/if}
	{/if}
</div>

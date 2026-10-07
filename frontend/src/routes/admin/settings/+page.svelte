<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { hasKey, i18n, LOCALES } from '$lib/i18n/index.svelte';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { fmtDateTime } from '$lib/i18n/format';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { onEvent } from '$lib/events.svelte';
	import LoaderIcon from '@lucide/svelte/icons/loader-circle';
	import PlayIcon from '@lucide/svelte/icons/play';
	import DownloadIcon from '@lucide/svelte/icons/download';
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

	// jobs this admin started from here: name -> when it last finished before the click, so the
	// next different finish time means "our run is over" and we can say how it went
	let started = $state<Record<string, number>>({});

	async function refreshSync() {
		try {
			sync = await unwrap(api.GET('/admin/sync'));
			for (const job of sync.jobs) {
				if (!(job.name in started) || job.running || job.lastFinishedAt === started[job.name]) continue;
				delete started[job.name];
				const name = jobName(job.name);
				const msg = job.message ? `: ${job.message}` : '';
				if (job.status === 'error') toast.error(t('settings.job_failed', { name, message: job.message || '' }));
				else if (job.status === 'skipped') toast.info(`${name} · ${t('job.skipped')}${msg}`);
				else toast.success(`${name} · ${t('settings.job_done')}${msg}`);
			}
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
		// the server pushes job changes; polling is the safety net while something runs
		const off = onEvent('sync.status', () => refreshSync());
		const timer = setInterval(() => {
			if (anyRunning || Object.keys(started).length) refreshSync();
		}, 1500);
		return () => {
			clearInterval(timer);
			off();
		};
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

	async function importLdap() {
		ldapBusy = true;
		try {
			const r = await unwrap(api.POST('/admin/ldap/import-jellyfin'));
			settings = await unwrap(api.GET('/admin/settings'));
			auth.avatarV++;
			if (r.connected) toast.success(t('settings.ldap_imported'));
			else toast.error(t('settings.ldap_imported_fail', { error: r.error ?? '' }));
		} catch (err) {
			toast.error(errorText(err));
		} finally {
			ldapBusy = false;
		}
	}

	// app-wide default language
	let defaultLang = $state('');
	$effect(() => {
		if (settings) defaultLang = settings.defaultLanguage;
	});
	async function saveDefaultLanguage(v: string) {
		defaultLang = v;
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { defaultLanguage: v } }));
			i18n.setAppDefault(v);
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
			settings = await unwrap(api.GET('/admin/settings'));
		}
	}

	async function toggleFolderChoice(on: boolean) {
		try {
			settings = await unwrap(api.PUT('/admin/settings', { body: { userFolderChoice: on } }));
			auth.userFolderChoice = on;
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
		started[job] = sync?.jobs.find((j) => j.name === job)?.lastFinishedAt ?? 0;
		try {
			await expectOk(api.POST('/admin/sync/{job}', { params: { path: { job } } })); // 202: nothing to read
			toast.info(t('settings.job_started', { name: jobName(job) }));
			await refreshSync();
		} catch (err) {
			delete started[job];
			toast.error(errorText(err));
		}
	}

	const jobName = (n: string) => (hasKey(`settings.job.${n}`) ? t(`settings.job.${n}` as 'settings.job_run') : n);
	const jobDesc = (n: string) => (hasKey(`settings.job.${n}.desc`) ? t(`settings.job.${n}.desc` as 'settings.job_run') : '');
	const jobMeta = (j: { status: string; message: string; lastStartedAt: number; lastFinishedAt: number }) =>
		[when(j.lastFinishedAt), j.lastStartedAt ? t('settings.job_took', { seconds: jobSeconds(j) }) : '', j.status === 'error' ? '' : j.message].filter(Boolean).join(' · ');
	const jobSeconds = (j: { lastStartedAt: number; lastFinishedAt: number }) => Math.max(0, j.lastFinishedAt - j.lastStartedAt);
	const every = (s: number) => (s >= 3600 ? t('settings.job_every', { hours: Math.round(s / 3600) }) : t('settings.job_every_min', { minutes: Math.max(1, Math.round(s / 60)) }));

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
					<div class="flex flex-wrap gap-2 sm:col-span-2">
						<Button type="submit" disabled={ldapBusy}>{ldapBusy ? t('common.saving') : t('common.save')}</Button>
						{#if settings.jellyfinApiKeyConfigured}
							<Button type="button" variant="outline" disabled={ldapBusy} onclick={importLdap}><DownloadIcon /> {t('settings.ldap_import')}</Button>
						{/if}
					</div>
				</form>
			</CardContent>
		</Card>

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
					{@const busy = job.running || job.name in started}
					<div class="grid gap-2 rounded-md border border-border p-3" aria-busy={busy}>
						<div class="flex items-start justify-between gap-3">
							<div class="grid min-w-0 gap-1">
								<div class="flex flex-wrap items-center gap-2">
									<span class="font-medium">{jobName(job.name)}</span>
									{#if busy}
										<Badge variant="outline" class="animate-pulse">{t('job.running')}</Badge>
									{:else if job.status === 'ok'}
										<Badge class="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">{statusLabel(job.status)}</Badge>
									{:else if job.status === 'error'}
										<Badge variant="destructive">{statusLabel(job.status)}</Badge>
									{:else}
										<Badge variant="secondary">{statusLabel(job.status)}</Badge>
									{/if}
								</div>
								{#if jobDesc(job.name)}<span class="text-xs text-muted-foreground">{jobDesc(job.name)}</span>{/if}
								{#if !busy && job.status !== 'never'}
									<span class="text-xs text-muted-foreground">{jobMeta(job)}</span>
									{#if job.status === 'error' && job.message}<span class="text-xs text-destructive">{job.message}</span>{/if}
								{/if}
								<span class="text-xs text-muted-foreground">{every(job.everySeconds)}</span>
							</div>
							<Button
								variant="outline"
								size="sm"
								class="shrink-0"
								disabled={(job.name !== 'boxoffice-refresh' && job.name !== 'servarr-import' && !sync.canSync) || busy}
								onclick={() => runJob(job.name as JobName)}
							>
								{#if busy}<LoaderIcon class="animate-spin" /> {t('settings.job_running_btn')}{:else}<PlayIcon /> {t('settings.job_run')}{/if}
							</Button>
						</div>
						{#if busy}
							<div class="h-1 overflow-hidden rounded-full bg-muted" role="progressbar" aria-label={t('settings.job_running_btn')}>
								<div class="job-bar h-full w-1/3 rounded-full bg-primary"></div>
							</div>
						{/if}
					</div>
				{/each}
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

<style>
	.job-bar {
		animation: job-slide 1.3s ease-in-out infinite;
	}
	@keyframes job-slide {
		0% {
			margin-left: -34%;
		}
		100% {
			margin-left: 100%;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.job-bar {
			animation: none;
			margin-left: 0;
			width: 100%;
			opacity: 0.5;
		}
	}
</style>

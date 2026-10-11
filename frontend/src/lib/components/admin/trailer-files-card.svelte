<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { fmtBytes, fmtDateTime } from '$lib/i18n/format';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import LoaderIcon from '@lucide/svelte/icons/loader-circle';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import RefreshIcon from '@lucide/svelte/icons/refresh-cw';

	const settings = $derived(admin.settings);
	const heights = ['360', '480', '720', '1080'];

	// the parent renders this card only once the settings are loaded
	let url = $state(admin.settings?.teasersFilesUrl ?? '');
	let key = $state('');
	let saving = $state(false);
	let overview = $state<Schemas['Overview'] | null>(null);
	let overviewError = $state<string | null>(null);
	let loading = $state(false);
	let testing = $state(false);
	let testResult = $state<{ ok: boolean; text: string } | null>(null);
	let clearing = $state(false);

	const configured = $derived(!!settings && !!settings.teasersFilesUrl && settings.teasersFilesKeyConfigured);
	const heightOptions = $derived(heights.map((h) => ({ value: h, label: `${h}p` })));

	async function loadOverview() {
		loading = true;
		overviewError = null;
		try {
			overview = await unwrap(api.GET('/admin/teasers/files'));
		} catch (e) {
			overviewError = errorText(e);
		} finally {
			loading = false;
		}
	}

	onMount(loadOverview);

	async function put(body: Schemas['UpdateSettingsInputBody']) {
		try {
			admin.settings = await unwrap(api.PUT('/admin/settings', { body }));
			toast.success(t('common.saved'));
			await loadOverview();
			return true;
		} catch (e) {
			toast.error(errorText(e));
			admin.settings = await unwrap(api.GET('/admin/settings'));
			return false;
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		const ok = await put({ teasersFilesUrl: url.trim(), ...(key.trim() ? { teasersFilesKey: key.trim() } : {}) });
		if (ok) key = '';
		url = admin.settings?.teasersFilesUrl ?? url;
		saving = false;
	}

	const toggle = (on: boolean) => put({ teasersFilesEnabled: on });
	const setHeight = (h: string) => put({ teasersFilesMaxHeight: Number(h) });

	async function runTest() {
		testing = true;
		testResult = null;
		try {
			const r = await unwrap(api.POST('/admin/teasers/files/test', { body: {} }));
			if (r.status === 'ready') {
				testResult = { ok: true, text: t('trailers.test_ok', { height: r.height, size: fmtBytes(r.bytes), seconds: Math.round(r.seconds) }) };
			} else if (r.status === 'failed') {
				testResult = { ok: false, text: t('trailers.test_failed', { error: r.error ?? '' }) };
			} else {
				testResult = { ok: false, text: t('trailers.test_pending', { status: r.status }) };
			}
		} catch (e) {
			testResult = { ok: false, text: t('trailers.test_failed', { error: errorText(e) }) };
		} finally {
			testing = false;
			await loadOverview();
		}
	}

	async function clearCache() {
		if (!confirm(t('trailers.clear_confirm'))) return;
		clearing = true;
		try {
			await expectOk(api.DELETE('/admin/teasers/files'));
			toast.success(t('trailers.cleared'));
		} catch (e) {
			toast.error(errorText(e));
		} finally {
			clearing = false;
			await loadOverview();
		}
	}

	const health = $derived(overview?.health);
	const files = $derived(overview?.files ?? {});
</script>

{#if settings}
	<Card>
		<CardHeader>
			<CardTitle>{t('trailers.title')}</CardTitle>
			<CardDescription>{t('trailers.desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-4">
			<div class="flex gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm" role="note">
				<TriangleAlertIcon class="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
				<p>{t('trailers.warning')}</p>
			</div>

			<form class="grid gap-2" onsubmit={save}>
				<Input bind:value={url} placeholder={t('trailers.url_placeholder')} type="url" autocomplete="off" aria-label={t('trailers.url_placeholder')} />
				<Input
					bind:value={key}
					placeholder={settings.teasersFilesKeyConfigured ? t('trailers.key_saved') : t('trailers.key_placeholder')}
					type="password"
					autocomplete="off"
					aria-label={t('trailers.key_placeholder')}
				/>
				<div>
					<Button type="submit" variant="outline" disabled={saving || (!key.trim() && url.trim() === settings.teasersFilesUrl)}>
						{saving ? t('common.saving') : t('common.save')}
					</Button>
				</div>
			</form>

			<div class="grid gap-1.5">
				<label class="flex cursor-pointer items-center gap-2 text-sm {!configured && !settings.teasersFilesEnabled ? 'cursor-not-allowed opacity-60' : ''}">
					<input
						type="checkbox"
						checked={settings.teasersFilesEnabled}
						disabled={!configured && !settings.teasersFilesEnabled}
						onchange={(e) => toggle(e.currentTarget.checked)}
					/>
					{t('trailers.toggle')}
				</label>
				{#if !configured && !settings.teasersFilesEnabled}<p class="text-xs text-muted-foreground">{t('trailers.need_config')}</p>{/if}
			</div>

			<div class="flex flex-wrap items-center gap-2 text-sm">
				<span class="text-muted-foreground">{t('trailers.height')}</span>
				<SimpleSelect label={t('trailers.height')} value={String(settings.teasersFilesMaxHeight)} options={heightOptions} onchange={setHeight} class="w-28" />
			</div>

			<div class="grid gap-2 rounded-md border border-border p-3 text-sm" aria-busy={loading}>
				<div class="flex items-center justify-between gap-2">
					<span class="font-medium">{t('trailers.status')}</span>
					<Button variant="ghost" size="sm" onclick={loadOverview} disabled={loading} aria-label={t('trailers.refresh')} title={t('trailers.refresh')}>
						<RefreshIcon class={loading ? 'animate-spin' : ''} />
					</Button>
				</div>
				{#if overviewError}
					<p class="text-destructive">{overviewError}</p>
				{:else if overview}
					{#if !overview.configured}
						<p class="text-muted-foreground">{t('trailers.not_configured')}</p>
					{:else}
						{#if !overview.enabled}<p class="text-xs text-muted-foreground">{t('trailers.off_hint')}</p>{/if}
						<div class="flex items-center justify-between border-b border-border py-1.5">
							<span class="text-muted-foreground">{t('trailers.reachable')}</span>
							{#if overview.available}
								<Badge class="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">{t('trailers.reachable_yes')}</Badge>
							{:else}
								<Badge variant="destructive">{t('trailers.reachable_no')}</Badge>
							{/if}
						</div>
						{#if overview.error && !overview.available}<p class="text-xs text-destructive">{overview.error}</p>{/if}
						{#if health}
							<div class="flex flex-wrap justify-between gap-x-4 border-b border-border py-1.5">
								<span class="text-muted-foreground">{t('trailers.versions')}</span>
								<span class="font-mono text-xs break-all">yt-dlp {health.ytdlpVersion || '?'} · Deno {health.denoVersion.split(' ')[1] || health.denoVersion || '?'} · ffmpeg {health.ffmpegVersion.split(' ')[2] || health.ffmpegVersion || '?'}</span>
							</div>
							<div class="flex justify-between gap-4 border-b border-border py-1.5">
								<span class="text-muted-foreground">{t('trailers.cache')}</span>
								<span>{t('trailers.cache_value', { used: fmtBytes(health.cacheBytes), limit: fmtBytes(health.cacheLimitBytes) })}</span>
							</div>
							<div class="flex justify-between gap-4 border-b border-border py-1.5">
								<span class="text-muted-foreground">{t('trailers.downloads')}</span>
								<span class={health.breaker.state === 'open' ? 'text-destructive' : ''}>
									{health.breaker.state === 'open' ? t('trailers.breaker_open', { time: fmtDateTime(health.breaker.until) }) : t('trailers.breaker_closed')}
								</span>
							</div>
							{#if health.lastError}
								<div class="grid gap-0.5 py-1.5">
									<span class="text-muted-foreground">{t('trailers.last_error')}</span>
									<span class="text-xs break-words text-destructive">{health.lastError}</span>
								</div>
							{/if}
						{/if}
						<div class="flex justify-between gap-4 py-1.5">
							<span class="text-muted-foreground">{t('trailers.files')}</span>
							<span>{t('trailers.files_value', { ready: files.ready ?? 0, queued: files.queued ?? 0, running: files.running ?? 0, failed: files.failed ?? 0 })}</span>
						</div>
					{/if}
				{:else}
					<p class="text-muted-foreground">{t('common.loading')}</p>
				{/if}
			</div>

			<div class="flex flex-wrap items-center gap-2">
				<Button variant="outline" onclick={runTest} disabled={testing || !configured}>
					{#if testing}<LoaderIcon class="animate-spin" />{/if}
					{t('trailers.test')}
				</Button>
				<Button variant="outline" onclick={clearCache} disabled={clearing || !configured}>
					{#if clearing}<LoaderIcon class="animate-spin" />{/if}
					{t('trailers.clear')}
				</Button>
			</div>
			{#if testing}
				<div class="grid gap-1.5" role="status">
					<p class="text-sm text-muted-foreground">{t('trailers.testing')}</p>
					<div class="h-1 overflow-hidden rounded-full bg-muted" role="progressbar" aria-label={t('trailers.testing')}>
						<div class="test-bar h-full w-1/3 rounded-full bg-primary"></div>
					</div>
				</div>
			{:else if testResult}
				<p class="text-sm {testResult.ok ? 'text-emerald-600 dark:text-emerald-400' : 'text-destructive'}" role="status">{testResult.text}</p>
			{/if}
		</CardContent>
	</Card>
{/if}

<style>
	.test-bar {
		animation: test-slide 1.3s ease-in-out infinite;
	}
	@keyframes test-slide {
		0% {
			margin-left: -34%;
		}
		100% {
			margin-left: 100%;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.test-bar {
			animation: none;
		}
	}
</style>

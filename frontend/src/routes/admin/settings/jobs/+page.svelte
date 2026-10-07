<script lang="ts">
	import { t, hasKey } from '$lib/i18n/index.svelte';
	import { api, expectOk, errorText } from '$lib/api/client';
	import { admin } from '$lib/stores/admin-settings.svelte';
	import { toast } from '$lib/toast.svelte';
	import { fmtDateTime } from '$lib/i18n/format';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import LoaderIcon from '@lucide/svelte/icons/loader-circle';
	import PlayIcon from '@lucide/svelte/icons/play';

	const settings = $derived(admin.settings);
	const sync = $derived(admin.sync);

	type JobName = 'library-sync' | 'history-sync' | 'boxoffice-refresh' | 'servarr-import';

	async function runJob(job: JobName) {
		admin.error = null;
		admin.started[job] = sync?.jobs.find((j) => j.name === job)?.lastFinishedAt ?? 0;
		try {
			await expectOk(api.POST('/admin/sync/{job}', { params: { path: { job } } })); // 202: nothing to read
			toast.info(t('settings.job_started', { name: admin.jobName(job) }));
			await admin.refreshSync();
		} catch (err) {
			delete admin.started[job];
			toast.error(errorText(err));
		}
	}

	const jobName = admin.jobName;
	const jobDesc = (n: string) => (hasKey(`settings.job.${n}.desc`) ? t(`settings.job.${n}.desc` as 'settings.job_run') : '');
	const jobSeconds = (j: { lastStartedAt: number; lastFinishedAt: number }) => Math.max(0, j.lastFinishedAt - j.lastStartedAt);
	const jobMeta = (j: { status: string; message: string; lastStartedAt: number; lastFinishedAt: number }) =>
		[when(j.lastFinishedAt), j.lastStartedAt ? t('settings.job_took', { seconds: jobSeconds(j) }) : '', j.status === 'error' ? '' : j.message].filter(Boolean).join(' · ');
	const every = (s: number) => (s >= 3600 ? t('settings.job_every', { hours: Math.round(s / 3600) }) : t('settings.job_every_min', { minutes: Math.max(1, Math.round(s / 60)) }));
	const when = (unix: number) => (unix ? fmtDateTime(unix) : t('common.never'));
	const statusLabel = (s: string) => (hasKey(`job.${s}`) ? t(`job.${s}` as 'job.ok') : s);
	const webhookUrl = $derived(settings ? `${location.origin}${settings.webhookPath}` : '');
</script>

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
				{@const busy = job.running || job.name in admin.started}
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
{:else}
	<p class="text-sm text-muted-foreground">{t('common.loading')}</p>
{/if}

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

	<Card>
		<CardHeader>
			<CardTitle>{t('settings.outhooks_title')}</CardTitle>
			<CardDescription>{t('settings.outhooks_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<Button variant="outline" href="/admin/webhooks">{t('settings.outhooks_open')}</Button>
		</CardContent>
	</Card>
{/if}

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

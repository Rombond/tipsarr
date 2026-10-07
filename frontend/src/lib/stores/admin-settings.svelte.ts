// What the admin settings tabs share: the settings, the background-job status and the jobs this
// admin started from here (so the end of their run can be announced).
import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
import { hasKey, t } from '$lib/i18n/index.svelte';
import { toast } from '$lib/toast.svelte';

class AdminSettings {
	settings = $state<Schemas['SettingsBody'] | null>(null);
	sync = $state<Schemas['SyncStatusBody'] | null>(null);
	error = $state<string | null>(null);
	/** Jobs started from this page: name -> when it last finished before the click. */
	started = $state<Record<string, number>>({});

	get anyRunning() {
		return this.sync?.jobs.some((j) => j.running) ?? false;
	}

	jobName = (n: string) => (hasKey(`settings.job.${n}`) ? t(`settings.job.${n}` as 'settings.job_run') : n);

	async loadSettings() {
		try {
			this.settings = await unwrap(api.GET('/admin/settings'));
		} catch (e) {
			this.error = errorText(e);
		}
	}

	async refreshSync() {
		try {
			const sync = await unwrap(api.GET('/admin/sync'));
			this.sync = sync;
			for (const job of sync.jobs) {
				// our run is over once the job stopped and its finish time changed since the click
				if (!(job.name in this.started) || job.running || job.lastFinishedAt === this.started[job.name]) continue;
				delete this.started[job.name];
				const name = this.jobName(job.name);
				const msg = job.message ? `: ${job.message}` : '';
				if (job.status === 'error') toast.error(t('settings.job_failed', { name, message: job.message || '' }));
				else if (job.status === 'skipped') toast.info(`${name} · ${t('job.skipped')}${msg}`);
				else toast.success(`${name} · ${t('settings.job_done')}${msg}`);
			}
		} catch (e) {
			this.error = errorText(e);
		}
	}
}

export const admin = new AdminSettings();

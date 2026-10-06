<script lang="ts" module>
	export type FlowHandle = { start: () => void };
</script>

<script lang="ts">
	// One request flow for every "Request" button: shows the dialog when the person has a choice to
	// make (seasons of a show, or an admin picking a quality profile) and requests directly otherwise.
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import RequestDialog, { type RequestChoice } from './request-dialog.svelte';

	let {
		type,
		tmdbId,
		title = '',
		seasons = undefined,
		busy = $bindable(false),
		onrequested,
	}: {
		type: 'movie' | 'tv';
		tmdbId: number;
		title?: string;
		/** Known seasons; fetched on demand when omitted (cards). */
		seasons?: Schemas['Season'][];
		busy?: boolean;
		onrequested?: (r: Schemas['View']) => void;
	} = $props();

	let open = $state(false);
	let error = $state<string | null>(null);
	let loaded = $state<Schemas['Season'][] | null>(null);
	const known = $derived(seasons ?? loaded ?? []);

	async function submit(choice: RequestChoice) {
		busy = true;
		error = null;
		try {
			const r = await unwrap(
				api.POST('/requests', { body: { type, tmdbId, seasons: choice.seasons, qualityProfileId: choice.qualityProfileId, rootFolder: choice.rootFolder } }),
			);
			open = false;
			toast.success(
				r.status === 'approved'
					? title ? t(r.dryRun ? 'media.toast_approved_dry' : 'media.toast_approved', { title }) : t('req.toast_approved_generic')
					: title ? t('media.toast_requested', { title }) : t('req.toast_requested_generic'),
			);
			onrequested?.(r);
		} catch (e) {
			error = errorText(e);
			if (!open) toast.error(error);
		} finally {
			busy = false;
		}
	}

	export async function start() {
		error = null;
		if (type === 'tv' && seasons === undefined && loaded === null) {
			busy = true;
			try {
				loaded = (await unwrap(api.GET('/media/{type}/{id}', { params: { path: { type: 'tv', id: tmdbId } } }))).seasons ?? [];
			} catch (e) {
				toast.error(errorText(e));
				busy = false;
				return;
			}
			busy = false;
		}
		const regular = known.filter((s) => s.number > 0);
		if (auth.isAdmin || (type === 'tv' && regular.length > 1)) open = true;
		else submit(type === 'tv' ? { seasons: regular.map((s) => s.number) } : {});
	}
</script>

<RequestDialog bind:open {type} {title} seasons={known} {busy} {error} onsubmit={submit} />

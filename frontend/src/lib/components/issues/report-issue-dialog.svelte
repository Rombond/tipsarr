<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { toast } from '$lib/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	type Kind = 'video' | 'audio' | 'subtitles' | 'other';

	let {
		open = $bindable(false),
		type,
		tmdbId,
		title,
		seasons = [],
	}: { open?: boolean; type: 'movie' | 'tv'; tmdbId: number; title: string; seasons?: Schemas['Season'][] } = $props();

	let kind = $state<Kind>('video');
	let season = $state('0');
	let episode = $state('');
	let message = $state('');
	let busy = $state(false);
	let error = $state<string | null>(null);

	const kinds: Kind[] = ['video', 'audio', 'subtitles', 'other'];
	const seasonOptions = $derived([
		{ value: '0', label: t('issue.whole_show') },
		...seasons.filter((s) => s.number > 0).map((s) => ({ value: String(s.number), label: s.name || t('seasons.season', { n: s.number }) })),
	]);

	$effect(() => {
		if (open) {
			kind = 'video';
			season = '0';
			episode = '';
			message = '';
			error = null;
		}
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			const issue = await unwrap(
				api.POST('/issues', {
					body: {
						type,
						tmdbId,
						kind,
						message: message.trim(),
						...(type === 'tv' ? { season: Number(season), episode: season === '0' ? 0 : Number(episode) || 0 } : {}),
					},
				}),
			);
			open = false;
			toast.success(t('issue.toast_reported'));
			goto(`/issues/${issue.id}`);
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>{t('issue.report_title', { title })}</Dialog.Title>
			<Dialog.Description>{t('issue.report_desc')}</Dialog.Description>
		</Dialog.Header>
		<form class="grid gap-4" onsubmit={submit}>
			<fieldset class="grid gap-1.5">
				<legend class="mb-1.5 text-sm font-medium">{t('issue.what')}</legend>
				<div class="flex flex-wrap gap-1.5">
					{#each kinds as k (k)}
						<label
							class="cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary has-[:checked]:text-primary-foreground has-[:focus-visible]:ring-[3px] has-[:focus-visible]:ring-ring/50 {kind === k ? '' : 'border-border text-muted-foreground hover:bg-accent'}"
						>
							<input type="radio" name="kind" value={k} class="sr-only" checked={kind === k} onchange={() => (kind = k)} />
							{t(`issue.kind.${k}` as 'issue.kind.video')}
						</label>
					{/each}
				</div>
			</fieldset>
			{#if type === 'tv' && seasons.length}
				<div class="grid grid-cols-2 gap-3">
					<div class="grid gap-1.5 text-sm">
						<span class="font-medium">{t('issue.which_season')}</span>
						<SimpleSelect label={t('issue.which_season')} value={season} options={seasonOptions} onchange={(v) => (season = v)} class="w-full" />
					</div>
					{#if season !== '0'}
						<label class="grid gap-1.5 text-sm">
							<span class="font-medium">{t('issue.episode')}</span>
							<input
								type="number"
								min="1"
								bind:value={episode}
								placeholder={t('issue.episode_placeholder')}
								class="h-9 rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
							/>
						</label>
					{/if}
				</div>
			{/if}
			<label class="grid gap-1.5 text-sm">
				<span class="font-medium">{t('issue.describe')}</span>
				<textarea
					bind:value={message}
					rows="4"
					maxlength="2000"
					required
					placeholder={t('issue.describe_placeholder')}
					class="rounded-md border border-input bg-background px-3 py-2 text-sm shadow-xs outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
				></textarea>
			</label>
			{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => (open = false)}>{t('common.cancel')}</Button>
				<Button type="submit" disabled={busy || !message.trim()}>{busy ? t('common.saving') : t('issue.submit')}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

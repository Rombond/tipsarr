<script lang="ts">
	const DOCS_URL = 'https://github.com/Rombond/tipsarr/blob/master/docs/notifications.md';
	import { t } from '$lib/i18n/index.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	type Hook = Schemas['WebhookView'];
	type Preset = Schemas['Preset'];
	type Header = Schemas['Header'];
	type AuthType = 'none' | 'bearer' | 'basic';

	let hooks = $state<Hook[]>([]);
	let presets = $state<Preset[]>([]);
	let allEvents = $state<string[]>([]);
	let error = $state<string | null>(null);
	let busy = $state(false);
	let publicUrl = $state('');

	// the form
	let editingId = $state<string | null>(null); // 'new' or a webhook id
	let editing = $derived(hooks.find((h) => h.id === editingId) ?? null);
	let preset = $state('raw');
	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let events = $state<string[]>([]);
	let enabled = $state(true);
	let template = $state('');
	let headers = $state<Header[]>([]);
	let authType = $state<AuthType>('none');
	let authUsername = $state('');
	let authSecret = $state('');
	let perRecipient = $state(false);

	// preview
	let previewEvent = $state('request.created');
	let preview = $state<Schemas['WebhookPreview'] | null>(null);

	const presetHint = $derived(presets.find((p) => p.id === preset));

	async function load() {
		try {
			hooks = await unwrap(api.GET('/admin/webhooks'));
		} catch (e) {
			error = errorText(e);
		}
	}

	onMount(() => {
		if (!auth.canManageSettings) {
			goto('/discover');
			return;
		}
		load();
		unwrap(api.GET('/admin/webhooks/presets'))
			.then((r) => {
				presets = r.presets;
				allEvents = r.events;
			})
			.catch((e) => (error = errorText(e)));
		unwrap(api.GET('/admin/settings'))
			.then((s) => (publicUrl = s.appPublicUrl))
			.catch(() => {});
	});

	function edit(h: Hook | null) {
		editingId = h?.id ?? 'new';
		preset = h?.preset || 'raw';
		name = h?.name ?? '';
		url = h?.url ?? '';
		secret = '';
		events = h ? [...h.events] : [];
		enabled = h?.enabled ?? true;
		template = h?.template ?? '';
		headers = h ? h.headers.map((x) => ({ ...x })) : [];
		authType = (h?.authType as AuthType) ?? 'none';
		authUsername = h?.authUsername ?? '';
		authSecret = '';
		perRecipient = h?.perRecipient ?? false;
		preview = null;
	}

	function applyPreset(id: string) {
		const p = presets.find((x) => x.id === id);
		if (!p) return;
		if (template.trim() && template !== presets.find((x) => x.id === preset)?.template && !confirm(t('webhooks.preset_replace'))) return;
		preset = id;
		template = p.template;
		headers = p.headers.map((x) => ({ ...x }));
		authType = p.authType as AuthType;
		if (!url.trim()) url = p.urlExample;
		if (!name.trim()) name = p.name;
		preview = null;
	}

	function body() {
		return {
			name: name.trim(),
			url: url.trim(),
			events,
			enabled,
			preset,
			template,
			headers: headers.filter((h) => h.name.trim()),
			authType,
			authUsername,
			perRecipient,
			...(secret ? { secret } : {}),
			...(authSecret ? { authSecret } : {})
		};
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			if (editingId === 'new') await unwrap(api.POST('/admin/webhooks', { body: body() }));
			else await unwrap(api.PUT('/admin/webhooks/{id}', { params: { path: { id: editingId! } }, body: body() }));
			editingId = null;
			toast.success(t('common.saved'));
			await load();
		} catch (err) {
			error = errorText(err);
		} finally {
			busy = false;
		}
	}

	async function runPreview() {
		error = null;
		try {
			const id = editingId && editingId !== 'new' ? editingId : undefined;
			preview = await unwrap(api.POST('/admin/webhooks/preview', { params: { query: { id, event: previewEvent } }, body: body() }));
		} catch (e) {
			error = errorText(e);
		}
	}

	async function test(h: Hook) {
		error = null;
		try {
			await expectOk(api.POST('/admin/webhooks/{id}/test', { params: { path: { id: h.id } } }));
			toast.success(t('webhooks.tested', { name: h.name }));
		} catch (e) {
			error = errorText(e);
		}
	}

	async function remove(h: Hook) {
		if (!confirm(t('webhooks.delete_confirm', { name: h.name }))) return;
		try {
			await expectOk(api.DELETE('/admin/webhooks/{id}', { params: { path: { id: h.id } } }));
			await load();
		} catch (e) {
			error = errorText(e);
		}
	}

	async function savePublicUrl(e: SubmitEvent) {
		e.preventDefault();
		try {
			const s = await unwrap(api.PUT('/admin/settings', { body: { appPublicUrl: publicUrl.trim() } }));
			publicUrl = s.appPublicUrl;
			toast.success(t('common.saved'));
		} catch (err) {
			toast.error(errorText(err));
		}
	}

	function toggle(ev: string) {
		events = events.includes(ev) ? events.filter((x) => x !== ev) : [...events, ev];
	}
</script>

<svelte:head>
	<title>{t('webhooks.title')} · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
	<h1 class="font-bold text-3xl">{t('webhooks.title')}</h1>
	<p class="text-sm text-muted-foreground">{t('webhooks.intro')}</p>
	{#if error}<p class="text-sm text-destructive">{error}</p>{/if}

	<Card>
		<CardHeader>
			<CardTitle>{t('webhooks.public_url_title')}</CardTitle>
			<CardDescription>{t('webhooks.public_url_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="flex gap-2" onsubmit={savePublicUrl}>
				<Input bind:value={publicUrl} type="url" placeholder="https://tipsarr.example.org" autocomplete="off" />
				<Button type="submit" variant="outline">{t('common.save')}</Button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('webhooks.card_title')}</CardTitle>
			<CardDescription>{t('webhooks.events_line', { events: allEvents.join(', ') })}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-3">
			{#each hooks as h (h.id)}
				<div class="flex items-center justify-between gap-3 rounded-md border border-border p-3 text-sm">
					<div class="grid min-w-0 gap-0.5">
						<span class="flex flex-wrap items-center gap-2 font-medium">
							{h.name}
							{#if h.preset && h.preset !== 'raw'}<Badge variant="secondary">{presets.find((p) => p.id === h.preset)?.name ?? h.preset}</Badge>{/if}
							{#if !h.enabled}<Badge variant="outline">{t('webhooks.disabled')}</Badge>{/if}
							{#if h.secretConfigured}<Badge variant="secondary">{t('webhooks.signed')}</Badge>{/if}
							{#if h.perRecipient}<Badge variant="secondary">{t('webhooks.per_recipient_badge')}</Badge>{/if}
						</span>
						<span class="truncate font-mono text-xs text-muted-foreground">{h.url}</span>
						<span class="text-xs text-muted-foreground">{h.events.length ? h.events.join(', ') : t('webhooks.all_events')}</span>
					</div>
					<div class="flex shrink-0 gap-2">
						<Button size="sm" variant="outline" onclick={() => test(h)}>{t('common.test')}</Button>
						<Button size="sm" variant="outline" onclick={() => edit(h)}>{t('common.edit')}</Button>
						<Button size="sm" variant="ghost" onclick={() => remove(h)}>{t('common.delete')}</Button>
					</div>
				</div>
			{/each}

			{#if editingId}
				<form class="grid gap-4 rounded-lg border border-border p-3" onsubmit={save}>
					<div class="grid gap-1.5 text-sm">
						{t('webhooks.preset')}
						<SimpleSelect label={t('webhooks.preset')} value={preset} options={presets.map((p) => ({ value: p.id, label: p.name }))} onchange={applyPreset} class="w-full max-w-xs" />
						{#if presetHint?.notes}<span class="text-xs text-muted-foreground">{presetHint.notes}</span>{/if}
					</div>
					<label class="grid gap-1.5 text-sm">
						{t('webhooks.name')}
						<Input bind:value={name} required maxlength={100} />
					</label>
					<label class="grid gap-1.5 text-sm">
						{t('webhooks.url_label')}
						<Input bind:value={url} required placeholder={presetHint?.urlExample ?? 'https://'} class="font-mono" />
						<span class="text-xs text-muted-foreground">{t('webhooks.url_hint', { example: '{{ media.type }}' })}</span>
					</label>

					<div class="grid gap-1.5 text-sm">
						{t('webhooks.template')}
						<textarea
							bind:value={template}
							rows="12"
							spellcheck="false"
							class="w-full rounded-md border border-input bg-transparent p-2 font-mono text-xs"
							placeholder={t('webhooks.template_placeholder')}
						></textarea>
						<span class="text-xs text-muted-foreground">{t('webhooks.template_hint', { example: '{{ message | json }}' })}</span>
						<a class="text-xs text-primary underline-offset-4 hover:underline" href={DOCS_URL} target="_blank" rel="noopener noreferrer">{t('webhooks.docs_link')} ↗</a>
					</div>

					<div class="grid gap-1.5 text-sm">
						{t('webhooks.auth')}
						<SimpleSelect
							label={t('webhooks.auth')}
							value={authType}
							options={[
								{ value: 'none', label: t('webhooks.auth_none') },
								{ value: 'bearer', label: t('webhooks.auth_bearer') },
								{ value: 'basic', label: t('webhooks.auth_basic') }
							]}
							onchange={(v) => (authType = v as AuthType)}
							class="w-full max-w-xs"
						/>
						{#if authType === 'basic'}
							<Input bind:value={authUsername} placeholder={t('webhooks.auth_username')} autocomplete="off" />
						{/if}
						{#if authType !== 'none'}
							<Input
								bind:value={authSecret}
								type="password"
								autocomplete="new-password"
								placeholder={editing?.authSecretConfigured ? t('webhooks.auth_secret_keep') : authType === 'bearer' ? t('webhooks.auth_token') : t('webhooks.auth_password')}
							/>
						{/if}
					</div>

					<div class="grid gap-1.5 text-sm">
						{t('webhooks.headers')}
						{#each headers as h, i (i)}
							<div class="flex gap-2">
								<Input bind:value={h.name} placeholder={t('webhooks.header_name')} class="font-mono" />
								<Input bind:value={h.value} placeholder={t('webhooks.header_value')} class="font-mono" />
								<Button type="button" variant="ghost" size="sm" onclick={() => headers.splice(i, 1)}>{t('common.delete')}</Button>
							</div>
						{/each}
						<div><Button type="button" variant="outline" size="sm" onclick={() => headers.push({ name: '', value: '' })}>{t('webhooks.header_add')}</Button></div>
					</div>

					<Input placeholder={editing?.secretConfigured ? t('webhooks.secret_keep') : t('webhooks.secret_new')} bind:value={secret} autocomplete="off" />

					<div class="grid gap-1.5 text-sm">
						<span class="text-xs text-muted-foreground">{t('webhooks.events_hint')}</span>
						{#each allEvents as ev (ev)}
							<label class="flex items-center gap-2"><input type="checkbox" checked={events.includes(ev)} onchange={() => toggle(ev)} /> {ev}</label>
						{/each}
					</div>
					<label class="flex items-start gap-2 text-sm">
						<input type="checkbox" bind:checked={perRecipient} class="mt-1" />
						<span>{t('webhooks.per_recipient')}<br /><span class="text-xs text-muted-foreground">{t('webhooks.per_recipient_hint')}</span></span>
					</label>
					<label class="flex items-center gap-2 text-sm"><input type="checkbox" bind:checked={enabled} /> {t('webhooks.enabled')}</label>

					<div class="grid gap-2 rounded-md border border-dashed border-border p-3 text-sm">
						<div class="flex flex-wrap items-center gap-2">
							<SimpleSelect label={t('webhooks.preview_event')} value={previewEvent} options={['test', ...allEvents].map((e) => ({ value: e, label: e }))} onchange={(v) => (previewEvent = v)} class="w-56" />
							<Button type="button" variant="outline" size="sm" onclick={runPreview}>{t('webhooks.preview')}</Button>
						</div>
						{#if preview}
							{#if preview.error}
								<p class="text-destructive">{t('webhooks.preview_error', { error: preview.error })}</p>
							{:else}
								<p class="font-mono text-xs break-all">POST {preview.url}</p>
								<pre class="overflow-x-auto text-xs text-muted-foreground">{preview.headers.map((h) => `${h.name}: ${h.value}`).join('\n')}</pre>
								{#if !preview.validJson}<p class="text-xs text-destructive">{t('webhooks.preview_not_json')}</p>{/if}
								<pre class="max-h-80 overflow-auto rounded bg-muted p-2 text-xs">{preview.body}</pre>
							{/if}
						{/if}
					</div>

					<div class="flex gap-2">
						<Button type="submit" disabled={busy}>{busy ? t('common.saving') : t('common.save')}</Button>
						<Button type="button" variant="outline" onclick={() => (editingId = null)}>{t('common.cancel')}</Button>
					</div>
				</form>
			{:else}
				<div><Button size="sm" variant="outline" onclick={() => edit(null)}>{t('webhooks.add')}</Button></div>
			{/if}
		</CardContent>
	</Card>
</div>

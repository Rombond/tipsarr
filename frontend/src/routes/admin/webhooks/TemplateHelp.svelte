<script lang="ts">
	import { t, i18n } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText, type Schemas } from '$lib/api/client';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';

	type Variable = Schemas['Variable'];

	let { event = $bindable(), events, oninsert }: { event: string; events: string[]; oninsert: (snippet: string) => void } = $props();

	let vars = $state<Variable[]>([]);
	let error = $state<string | null>(null);
	let query = $state('');

	const shown = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return vars;
		return vars.filter((v) => v.path.toLowerCase().includes(q) || v.description.toLowerCase().includes(q));
	});

	$effect(() => {
		const ev = event;
		const lang = i18n.locale;
		let stale = false;
		unwrap(api.GET('/admin/webhooks/variables', { params: { query: { event: ev, lang } } }))
			.then((r) => {
				if (stale) return;
				vars = r;
				error = null;
			})
			.catch((e) => {
				if (!stale) error = errorText(e);
			});
		return () => {
			stale = true;
		};
	});

	// code samples stay out of the locale strings: t() reads " | " as a plural separator
	const cheats: { code: string; key: Parameters<typeof t>[0] }[] = [
		{ code: '{{ media.title }}', key: 'webhooks.help_output' },
		{ code: '{{ message | json }}', key: 'webhooks.help_json' },
		{ code: '{{ media.year | default: "?" }}', key: 'webhooks.help_default' },
		{ code: '{{ message | truncate: 50 }}', key: 'webhooks.help_truncate' },
		{ code: '{{ media.type | upcase }}', key: 'webhooks.help_upcase' },
		{ code: '{{ media.title | url_encode }}', key: 'webhooks.help_url_encode' },
		{ code: '{% if issue.id %}…{% endif %}', key: 'webhooks.help_if' },
		{ code: '{{ nl }}', key: 'webhooks.help_nl' }
	];
</script>

<details class="group rounded-md border border-border text-sm" open>
	<summary class="cursor-pointer select-none px-3 py-2 font-medium">{t('webhooks.help_title')}</summary>
	<div class="grid gap-3 border-t border-border p-3">
		<div class="flex flex-wrap items-center gap-2">
			<SimpleSelect label={t('webhooks.help_event')} value={event} options={['test', ...events].map((e) => ({ value: e, label: e }))} onchange={(v) => (event = v)} class="w-full sm:w-56" />
			<Input bind:value={query} type="search" placeholder={t('webhooks.help_search')} aria-label={t('webhooks.help_search')} class="min-w-0 flex-1" />
		</div>
		<p class="text-xs text-muted-foreground">{t('webhooks.help_click')}</p>
		{#if error}<p class="text-xs text-destructive">{error}</p>{/if}

		<ul class="grid max-h-80 gap-1 overflow-y-auto">
			{#each shown as v (v.path)}
				<li>
					<button
						type="button"
						class="grid w-full gap-0.5 rounded px-2 py-1.5 text-left hover:bg-muted focus-visible:bg-muted focus-visible:outline-none {v.empty ? 'opacity-60' : ''}"
						onclick={() => oninsert(`{{ ${v.path} }}`)}
					>
						<span class="flex flex-wrap items-center gap-1.5">
							<code class="font-mono text-xs font-semibold">{v.path}</code>
							<span class="text-[10px] text-muted-foreground">{v.type}</span>
							{#if v.empty}<Badge variant="outline">{t('webhooks.help_empty')}</Badge>{/if}
							{#if v.perRecipient}<Badge variant="secondary">{t('webhooks.help_per_recipient_badge')}</Badge>{/if}
						</span>
						{#if v.description}<span class="text-xs text-muted-foreground">{v.description}</span>{/if}
						{#if !v.empty}<span class="truncate font-mono text-[11px]">{JSON.stringify(v.example)}</span>{/if}
					</button>
				</li>
			{:else}
				<li class="px-2 text-xs text-muted-foreground">{t('webhooks.help_none')}</li>
			{/each}
		</ul>

		<div class="grid gap-1">
			<p class="font-medium">{t('webhooks.help_cheatsheet')}</p>
			<ul class="grid gap-1 text-xs text-muted-foreground">
				{#each cheats as c (c.code)}
					<li><code class="font-mono text-foreground">{c.code}</code> {t(c.key)}</li>
				{/each}
			</ul>
		</div>

		<div class="grid gap-1">
			<p class="font-medium">{t('webhooks.auth')}</p>
			<ul class="grid gap-1 text-xs text-muted-foreground">
				<li>{t('webhooks.help_auth_none')}</li>
				<li>{t('webhooks.help_auth_bearer', { header: 'Authorization: Bearer …' })}</li>
				<li>{t('webhooks.help_auth_basic', { header: 'Authorization: Basic …' })}</li>
				<li>{t('webhooks.help_per_recipient')}</li>
			</ul>
		</div>
	</div>
</details>

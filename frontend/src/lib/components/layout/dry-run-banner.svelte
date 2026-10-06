<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import FlaskIcon from '@lucide/svelte/icons/flask-conical';
</script>

{#if auth.dryRun}
	<div
		class="flex items-center justify-center gap-1.5 border-b border-amber-500/30 bg-amber-500/10 px-3 py-1 text-center text-[11px] text-amber-700 dark:text-amber-300"
		title={t('banner.dry_run_hint')}
	>
		<FlaskIcon class="size-3.5 shrink-0" />
		<span><strong>{t('banner.dry_run')}</strong> <span class="hidden sm:inline">· {t('banner.dry_run_long')}</span></span>
	</div>
{/if}

<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import FlaskIcon from '@lucide/svelte/icons/flask-conical';
</script>

{#if auth.dryRun}
	<div
		class="flex items-center justify-center gap-1.5 border-b border-amber-500/30 bg-amber-500/10 px-3 py-1 text-center text-[11px] text-amber-700 dark:text-amber-300"
		title="Requests are recorded and move through approval, but nothing is sent to Radarr or Sonarr. Set TIPSARR_DRY_RUN=false to go live."
	>
		<FlaskIcon class="size-3.5 shrink-0" />
		<span><strong>Dry-run</strong> <span class="hidden sm:inline">· nothing is sent to Radarr or Sonarr</span></span>
	</div>
{/if}

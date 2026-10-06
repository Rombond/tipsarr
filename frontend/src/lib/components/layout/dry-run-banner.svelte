<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
</script>

{#if auth.dryRun}
	<div class="border-b border-amber-500/40 bg-amber-500/10 px-4 py-1.5 text-center text-xs text-amber-700 dark:text-amber-300">
		Dry-run mode: nothing is ever sent to Radarr or Sonarr.
	</div>
{/if}

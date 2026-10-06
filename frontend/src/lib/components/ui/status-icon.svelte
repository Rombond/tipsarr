<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { STATUS, type StatusKey } from '$lib/status';

	let { status, class: className = '' }: { status: StatusKey; class?: string } = $props();
	const s = $derived(STATUS[status]);
	const label = $derived(t(`status.${status}` as 'status.available'));
</script>

<!-- a round icon; hovering (or focusing) it unfolds the state's name -->
<span
	class="group/status inline-flex h-6 items-center rounded-full px-1.5 shadow-md ring-1 ring-black/20 {s.solid} {className}"
	title={label}
	role="img"
	aria-label={label}
>
	<s.icon class="size-3.5 shrink-0" strokeWidth={3} />
	<span class="max-w-0 overflow-hidden whitespace-nowrap text-[11px] font-semibold opacity-0 transition-all duration-200 group-hover/status:ml-1 group-hover/status:max-w-28 group-hover/status:opacity-100">{label}</span>
</span>

<script lang="ts">
	// A themed replacement for native <select>: native popups ignore the app theme.
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import ChevronIcon from '@lucide/svelte/icons/chevron-down';
	import { cn } from '$lib/utils';

	type Option = { value: string; label: string };

	let {
		value,
		options,
		onchange,
		placeholder = 'Select…',
		label,
		class: className,
		disabled = false,
	}: {
		value: string;
		options: Option[];
		onchange: (value: string) => void;
		placeholder?: string;
		label?: string;
		class?: string;
		disabled?: boolean;
	} = $props();

	const current = $derived(options.find((o) => o.value === value));
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger {disabled}>
		{#snippet child({ props })}
			<button
				type="button"
				{...props}
				aria-label={label}
				class={cn(
					'flex h-9 min-w-24 cursor-pointer items-center justify-between gap-2 rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50',
					className,
				)}
			>
				<span class="truncate">{current?.label ?? placeholder}</span>
				<ChevronIcon class="size-4 shrink-0 text-muted-foreground" />
			</button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="max-h-72 w-max min-w-[max(12rem,var(--bits-dropdown-menu-anchor-width))] overflow-y-auto">
		<DropdownMenu.RadioGroup {value} onValueChange={(v) => onchange(v)}>
			{#each options as o (o.value)}
				<DropdownMenu.RadioItem value={o.value} class="py-1.5 whitespace-nowrap">{o.label}</DropdownMenu.RadioItem>
			{/each}
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>

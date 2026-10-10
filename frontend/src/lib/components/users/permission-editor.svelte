<script lang="ts">
	// The checkbox list of permission flags. Like in Seerr, a flag that brings others with it (manage requests brings
	// auto-approve, request brings movies and TV shows ...) switches them on and keeps them on while it is on.
	import { t, type Key } from '$lib/i18n/index.svelte';
	import { P, has, normalize, forced, type Flag } from '$lib/permissions';

	let {
		flags,
		mask = $bindable(0),
		disabled = false,
		keysLocked = false,
	}: {
		flags: Flag[];
		mask: number;
		/** read-only (an admin, or your own account) */
		disabled?: boolean;
		/** the keys to users and settings can only be given by admins */
		keysLocked?: boolean;
	} = $props();

	// flags marked reserved exist only so the numbers match Seerr: no feature uses them
	const shown = $derived(flags.filter((f) => !f.reserved));
	const groups = ['requests', 'auto-approve', 'issues', 'admin'];
	const locked = $derived(forced(mask, flags));

	function toggle(f: Flag, on: boolean) {
		const next = on ? mask | f.value : mask & ~f.value;
		mask = normalize(next, flags);
	}
	const isKey = (f: Flag) => f.value === P.manageUsers || f.value === P.manageSettings;
</script>

<div class="grid gap-4">
	{#each groups as group (group)}
		{@const items = shown.filter((f) => f.group === group)}
		{#if items.length}
			<fieldset class="grid gap-2">
				<legend class="mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t(`perm.group.${group}` as Key)}</legend>
				{#each items as f (f.name)}
					{@const off = disabled || (has(locked, f.value) && has(mask, f.value)) || (keysLocked && isKey(f))}
					<label class="flex items-start gap-3 text-sm {off ? 'opacity-70' : 'cursor-pointer'}">
						<input type="checkbox" class="mt-1" checked={has(mask, f.value)} disabled={off} onchange={(e) => toggle(f, e.currentTarget.checked)} />
						<span class="grid">
							<span class="font-medium">{t(`perm.${f.name}` as Key)}</span>
							<span class="text-xs text-muted-foreground">{t(`perm.${f.name}.desc` as Key)}</span>
						</span>
					</label>
				{/each}
			</fieldset>
		{/if}
	{/each}
</div>

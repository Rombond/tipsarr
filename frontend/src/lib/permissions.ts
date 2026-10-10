// Permission flags, the same values as backend/internal/perms (Seerr's numbers). A person's `effectivePermissions`
// is the sum of the flags they have; admins have every flag. GET /admin/permissions lists the flags for the editor.
export const P = {
	manageSettings: 4,
	manageUsers: 8,
	manageRequests: 16,
	request: 32,
	autoApprove: 128,
	autoApproveMovie: 256,
	autoApproveTV: 512,
	requestAdvanced: 8192,
	requestView: 16384,
	requestMovie: 262144,
	requestTV: 524288,
	manageIssues: 1048576,
	viewIssues: 2097152,
	createIssues: 4194304,
} as const;

/** True when the mask has every one of the flags. */
export function has(mask: number, ...flags: number[]): boolean {
	// bit operations on numbers above 2^31 need BigInt: the masks reach 2^30 today, but stay safe
	const m = BigInt(mask);
	return flags.every((f) => (m & BigInt(f)) === BigInt(f));
}

export type Flag = {
	name: string;
	value: number;
	group: string;
	implies?: string[] | null;
	reserved: boolean;
};

/** Switches on the flags that come with another one, as the server does when it saves a mask. */
export function normalize(mask: number, flags: Flag[]): number {
	let m = BigInt(mask);
	const byName = new Map(flags.map((f) => [f.name, BigInt(f.value)]));
	for (const f of flags) {
		if ((m & BigInt(f.value)) === 0n) continue;
		for (const name of f.implies ?? []) m |= byName.get(name) ?? 0n;
	}
	return Number(m);
}

/** The flags that are on only because another flag implies them (they cannot be switched off alone). */
export function forced(mask: number, flags: Flag[]): number {
	let out = 0n;
	const byName = new Map(flags.map((f) => [f.name, BigInt(f.value)]));
	for (const f of flags) {
		if ((BigInt(mask) & BigInt(f.value)) === 0n) continue;
		for (const name of f.implies ?? []) out |= byName.get(name) ?? 0n;
	}
	return Number(out);
}

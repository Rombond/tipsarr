// Typed client for the Tipsarr backend. Types are generated from the backend OpenAPI
// spec (`make generate` -> schema.d.ts); never hand-edit schema.d.ts.
import createClient from 'openapi-fetch';
import type { components, paths } from './schema';

export type Schemas = components['schemas'];
export type MediaItem = Schemas['Item'];
export type MediaDetail = Schemas['Detail'];
export type Person = Schemas['Person'];
export type User = Schemas['User'];
export type Genre = Schemas['Genre'];

export const api = createClient<paths>({ baseUrl: '/api/v1', credentials: 'include' });

export class ApiError extends Error {
	constructor(
		message: string,
		readonly status: number,
	) {
		super(message);
	}
}

/** Unwraps an openapi-fetch result: returns data or throws an ApiError with the server's message. */
export async function unwrap<T>(
	result: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
	const { data, error, response } = await result;
	if (!response.ok || data === undefined) {
		const e = error as { detail?: string; title?: string } | undefined;
		throw new ApiError(e?.detail || e?.title || `${response.status} ${response.statusText}`, response.status);
	}
	return data;
}

/** Like unwrap, for endpoints that answer 204 No Content. */
export async function expectOk(result: Promise<{ error?: unknown; response: Response }>): Promise<void> {
	const { error, response } = await result;
	if (!response.ok) {
		const e = error as { detail?: string; title?: string } | undefined;
		throw new ApiError(e?.detail || e?.title || `${response.status} ${response.statusText}`, response.status);
	}
}

/** URL for a TMDB image served (and cached) by the backend. `path` is a TMDB path like "/abc.jpg". */
export function imageUrl(path: string | null | undefined, size = 'w342'): string | null {
	return path ? `/api/v1/images/tmdb/${size}${path}` : null;
}

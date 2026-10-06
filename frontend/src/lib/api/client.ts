// Typed client for the Tipsarr backend. Types are generated from the backend OpenAPI
// spec (`make generate` -> schema.d.ts); never hand-edit schema.d.ts.
import createClient from 'openapi-fetch';
import type { components, paths } from './schema';
import { hasKey, t } from '$lib/i18n/index.svelte';

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
		/** Stable machine-readable code from the server (see backend internal/api/errors.go). */
		readonly code: string = '',
	) {
		super(message);
	}
}

type ErrorBody = { detail?: string; title?: string; errors?: { location?: string; value?: unknown }[] };

function toApiError(error: unknown, response: Response): ApiError {
	const e = error as ErrorBody | undefined;
	const code = e?.errors?.find((d) => d.location === 'code')?.value;
	return new ApiError(e?.detail || e?.title || `${response.status} ${response.statusText}`, response.status, typeof code === 'string' ? code : '');
}

/** The message to show a person: translated from the server's error code when we know it, then by HTTP status, then the server's text. */
export function errorText(e: unknown): string {
	if (e instanceof ApiError) {
		const byCode = `error.${e.code}`;
		if (e.code && hasKey(byCode)) return t(byCode);
		const byStatus = `error.status_${e.status}`;
		if (e.status !== 422 && hasKey(byStatus)) return t(byStatus);
		return e.message;
	}
	return e instanceof Error ? e.message : String(e);
}

/** Unwraps an openapi-fetch result: returns data or throws an ApiError with the server's message. */
export async function unwrap<T>(
	result: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
	const { data, error, response } = await result;
	if (!response.ok || data === undefined) throw toApiError(error, response);
	return data;
}

/** Like unwrap, for endpoints that answer 204 No Content. */
export async function expectOk(result: Promise<{ error?: unknown; response: Response }>): Promise<void> {
	const { error, response } = await result;
	if (!response.ok) throw toApiError(error, response);
}

/** URL for a TMDB image served (and cached) by the backend. `path` is a TMDB path like "/abc.jpg". */
export function imageUrl(path: string | null | undefined, size = 'w342'): string | null {
	return path ? `/api/v1/images/tmdb/${size}${path}` : null;
}

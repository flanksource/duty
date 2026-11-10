import createClient, { type Client } from "openapi-fetch";
import type { paths } from "./types";

export interface ClientOptions {
  baseUrl: string;
  token?: string;
  headers?: Record<string, string>;
  fetch?: typeof fetch;
}

/**
 * Creates a typed client against the Mission Control PostgREST API.
 *
 * Example:
 *   const mc = createMissionControlClient({ baseUrl: "https://api.example.com", token: jwt });
 *   const { data, error } = await mc.GET("/config_items", { params: { query: { limit: "10" } } });
 */
export function createMissionControlClient(opts: ClientOptions): Client<paths> {
  const headers: Record<string, string> = { ...opts.headers };
  if (opts.token) {
    headers["Authorization"] = `Bearer ${opts.token}`;
  }
  return createClient<paths>({
    baseUrl: opts.baseUrl,
    headers,
    fetch: opts.fetch,
  });
}

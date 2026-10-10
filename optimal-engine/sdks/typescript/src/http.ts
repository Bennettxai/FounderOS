import { OptimalEngineError } from "./error.js";

export interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export class HttpClient {
  readonly baseUrl: string;
  private readonly defaultHeaders: Record<string, string>;

  constructor(baseUrl: string, apiKey?: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.defaultHeaders = {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...(apiKey !== undefined ? { Authorization: `Bearer ${apiKey}` } : {}),
    };
  }

  async request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const init: RequestInit = {
      method: opts.method ?? "GET",
      headers: { ...this.defaultHeaders, ...(opts.headers ?? {}) },
    };
    if (opts.body !== undefined) {
      init.body = JSON.stringify(opts.body);
    }
    if (opts.signal !== undefined) {
      init.signal = opts.signal;
    }
    const res = await fetch(url, init);

    if (!res.ok) {
      throw await OptimalEngineError.fromResponse(res);
    }

    // 204 No Content
    if (res.status === 204) {
      return undefined as unknown as T;
    }

    return res.json() as Promise<T>;
  }

  get<T>(
    path: string,
    opts?: Omit<RequestOptions, "method" | "body">,
  ): Promise<T> {
    return this.request<T>(path, { ...opts, method: "GET" });
  }

  post<T>(
    path: string,
    body?: unknown,
    opts?: Omit<RequestOptions, "method" | "body">,
  ): Promise<T> {
    return this.request<T>(path, { ...opts, method: "POST", body });
  }

  patch<T>(
    path: string,
    body?: unknown,
    opts?: Omit<RequestOptions, "method" | "body">,
  ): Promise<T> {
    return this.request<T>(path, { ...opts, method: "PATCH", body });
  }

  delete<T>(
    path: string,
    opts?: Omit<RequestOptions, "method" | "body">,
  ): Promise<T> {
    return this.request<T>(path, { ...opts, method: "DELETE" });
  }

  /**
   * Return a copy of the auth/identity headers (no Content-Type) for use in
   * SSE connections that are opened outside the normal request() flow.
   */
  authHeaders(): Record<string, string> {
    const { "Content-Type": _ct, Accept: _acc, ...rest } = this.defaultHeaders;
    return { ...rest };
  }

  /** Build a query string from a plain object, omitting undefined/null values. */
  static buildQuery(
    params: Record<string, string | number | boolean | undefined | null>,
  ): string {
    const entries = Object.entries(params).filter(
      ([, v]) => v !== undefined && v !== null,
    );
    if (entries.length === 0) return "";
    const qs = new URLSearchParams(
      entries.map(([k, v]) => [k, String(v)]),
    ).toString();
    return `?${qs}`;
  }
}

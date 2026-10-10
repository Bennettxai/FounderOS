import type { OptimalEngineErrorBody } from "./types.js";

export class OptimalEngineError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "OptimalEngineError";
    this.status = status;
    this.code = code;
  }

  static async fromResponse(res: Response): Promise<OptimalEngineError> {
    let body: OptimalEngineErrorBody = {};
    try {
      body = (await res.json()) as OptimalEngineErrorBody;
    } catch {
      // ignore JSON parse failures
    }
    const message =
      body.message ?? body.error ?? `HTTP ${res.status} ${res.statusText}`;
    const code = body.code ?? `HTTP_${res.status}`;
    return new OptimalEngineError(res.status, code, message);
  }
}

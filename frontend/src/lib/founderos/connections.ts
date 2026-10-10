/** GET /api/founderos/connections: the Connections board (spec 4.19). */
export type ConnectionState = 'connected' | 'not_configured' | 'error' | (string & {});
export type Connection = { id: string; name: string; kind: string; state: ConnectionState; detail: string };
export type ConnectionsBody = { connections: Connection[] };

export type ConnectionsSummary = { up: number; total: number; errors: number; notConfigured: number };

export function summarizeConnections(conns: Connection[]): ConnectionsSummary {
	return {
		up: conns.filter((c) => c.state === 'connected').length,
		total: conns.length,
		errors: conns.filter((c) => c.state === 'error').length,
		notConfigured: conns.filter((c) => c.state === 'not_configured').length
	};
}

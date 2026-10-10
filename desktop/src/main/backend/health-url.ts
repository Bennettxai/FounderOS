// The backend binds 127.0.0.1 only; "localhost" can resolve to ::1 in Electron's
// Node and be refused, so health probes dial the IPv4 loopback explicitly.
export function healthUrl(port: number): string {
  return `http://127.0.0.1:${port}/health`;
}

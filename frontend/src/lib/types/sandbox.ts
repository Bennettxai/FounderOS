export type SandboxStatus = 'none' | 'pending' | 'deploying' | 'running' | 'stopped' | 'failed' | 'removing';

export type HealthStatus = 'unknown' | 'healthy' | 'unhealthy' | 'degraded';

export interface SandboxInfo {
  app_id: string;
  app_name: string;
  user_id: string;
  container_id: string;
  status: SandboxStatus;
  port: number;
  url: string;
  image: string;
  created_at: string;
  started_at?: string;
  health_status: HealthStatus;
  app_type: string;
  error_message?: string;
}

export interface SandboxContainer {
	id: string;
	app_id: string;
	status: SandboxStatus;
	url?: string;
	port?: number;
	started_at?: string;
	stopped_at?: string;
	error_message?: string;
}

export interface SandboxConfig {
	timeout_minutes: number;
	memory_mb: number;
	cpu_cores: number;
	auto_restart: boolean;
}

export interface SandboxMetrics {
	cpu_usage: number;
	memory_usage: number;
	uptime_seconds: number;
	request_count: number;
}

const SANDBOX_STATUS_COLORS: Record<SandboxStatus, { text: string; bg: string; border: string }> = {
  none: { text: 'text-gray-500', bg: 'bg-gray-100', border: 'border-gray-300' },
  pending: { text: 'text-gray-500', bg: 'bg-gray-100', border: 'border-gray-300' },
  deploying: { text: 'text-yellow-500', bg: 'bg-yellow-100', border: 'border-yellow-500' },
  running: { text: 'text-green-500', bg: 'bg-green-100', border: 'border-green-500' },
  stopped: { text: 'text-gray-500', bg: 'bg-gray-100', border: 'border-gray-300' },
  failed: { text: 'text-red-500', bg: 'bg-red-100', border: 'border-red-500' },
  removing: { text: 'text-orange-500', bg: 'bg-orange-100', border: 'border-orange-500' },
};

export function getSandboxStatusColor(status: SandboxStatus): string {
	return SANDBOX_STATUS_COLORS[status]?.text ?? SANDBOX_STATUS_COLORS.pending.text;
}

export function getSandboxStatusBgColor(status: SandboxStatus): string {
	return SANDBOX_STATUS_COLORS[status]?.bg ?? SANDBOX_STATUS_COLORS.pending.bg;
}

export function getSandboxStatusBorderColor(status: SandboxStatus): string {
	return SANDBOX_STATUS_COLORS[status]?.border ?? SANDBOX_STATUS_COLORS.pending.border;
}

export function isSandboxActive(status: SandboxStatus): boolean {
	return status === 'running' || status === 'deploying';
}

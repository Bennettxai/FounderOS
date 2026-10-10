/**
 * Multi-Agent App Generation Types
 * Matches backend Go types from desktop/backend-go/internal/agent/types.go
 */

export type AgentType =
  | "orchestrator"
  | "frontend"
  | "backend"
  | "database"
  | "test";

export type AgentStatus =
  | "pending"
  | "starting"
  | "in_progress"
  | "completed"
  | "failed";

/** SSE event from /api/osa/module-instances/generate/:id/stream */
export interface ProgressEvent {
  type?: string; // Event type from backend: connected, generation_started, agent_progress, generation_complete, error
  task_id: string;
  agent_type: AgentType;
  status: AgentStatus;
  message: string;
  progress: number; // 0-100
  timestamp: string;
  phase?: string;
  data?: Record<string, unknown>;
}

/** UI representation of an agent in AgentProgressPanel */
export interface AgentCard {
  id: string;
  name: string;
  type: AgentType;
  status: AgentStatus;
  progress: number;
  message: string;
  icon: "Code" | "Server" | "Database" | "TestTube";
  color: "blue" | "green" | "purple" | "orange";
}

export interface AgentConfig {
  type: AgentType;
  name: string;
  icon: AgentCard["icon"];
  color: AgentCard["color"];
  description: string;
}

export const AGENT_CONFIGS: Record<AgentType, AgentConfig> = {
  orchestrator: {
    type: "orchestrator",
    name: "Orchestrator",
    icon: "Code",
    color: "blue",
    description: "Coordinating all agents",
  },
  frontend: {
    type: "frontend",
    name: "Frontend Agent",
    icon: "Code",
    color: "blue",
    description: "Building UI components and pages",
  },
  backend: {
    type: "backend",
    name: "Backend Agent",
    icon: "Server",
    color: "green",
    description: "Creating API endpoints and business logic",
  },
  database: {
    type: "database",
    name: "Database Agent",
    icon: "Database",
    color: "purple",
    description: "Designing schema and migrations",
  },
  test: {
    type: "test",
    name: "Test Agent",
    icon: "TestTube",
    color: "orange",
    description: "Writing tests and validating functionality",
  },
};

export interface AppGenerationRequest {
  template_id?: string; // Optional - for template-based generation
  app_name: string;
  description: string;
  config?: Record<string, unknown>; // Optional config parameters
  complexity?: "simple" | "standard" | "complex"; // For AI generative mode
}

export interface AppGenerationResponse {
  queue_item_id: string;
  status: "queued" | "in_progress" | "completed" | "failed";
  message?: string;
}

export function isTerminalStatus(status: AgentStatus): boolean {
  return status === "completed" || status === "failed";
}

// Status color mappings for Tailwind classes
const STATUS_COLORS = {
  pending: {
    text: "text-gray-500",
    border: "border-gray-300",
    badge: "bg-gray-100 text-gray-700",
  },
  starting: {
    text: "text-yellow-500",
    border: "border-yellow-500",
    badge: "bg-yellow-100 text-yellow-700",
  },
  in_progress: {
    text: "text-blue-500",
    border: "border-blue-500",
    badge: "bg-blue-100 text-blue-700",
  },
  completed: {
    text: "text-green-500",
    border: "border-green-500",
    badge: "bg-green-100 text-green-700",
  },
  failed: {
    text: "text-red-500",
    border: "border-red-500",
    badge: "bg-red-100 text-red-700",
  },
} as const;

export function getStatusColorClass(status: AgentStatus): string {
  return STATUS_COLORS[status]?.text ?? STATUS_COLORS.pending.text;
}

export function getBorderColorClass(status: AgentStatus): string {
  return STATUS_COLORS[status]?.border ?? STATUS_COLORS.pending.border;
}

export function getStatusBadgeClass(status: AgentStatus): string {
  return STATUS_COLORS[status]?.badge ?? STATUS_COLORS.pending.badge;
}

// Simulated activity messages for agent progress panel
const AGENT_MESSAGES: Record<AgentType, Record<string, string[]>> = {
  orchestrator: {
    in_progress: [
      "Coordinating agent tasks...",
      "Validating agent outputs...",
      "Synchronizing parallel work...",
    ],
  },
  frontend: {
    in_progress: [
      "Generating React components...",
      "Building page layouts...",
      "Adding responsive styles...",
      "Creating navigation structure...",
      "Implementing form validation...",
    ],
  },
  backend: {
    in_progress: [
      "Setting up API routes...",
      "Creating service layer...",
      "Implementing business logic...",
      "Adding error handling...",
      "Configuring middleware...",
    ],
  },
  database: {
    in_progress: [
      "Designing database schema...",
      "Creating migration files...",
      "Setting up indexes...",
      "Configuring relationships...",
      "Preparing migration files...",
    ],
  },
  test: {
    in_progress: [
      "Writing unit tests...",
      "Creating integration tests...",
      "Validating API endpoints...",
      "Testing edge cases...",
      "Checking error handling...",
    ],
  },
};

export function getRandomMessage(agentType: AgentType, status: string): string {
  const messages = AGENT_MESSAGES[agentType]?.[status];
  if (!messages || messages.length === 0) {
    return "Working...";
  }
  return messages[Math.floor(Math.random() * messages.length)];
}

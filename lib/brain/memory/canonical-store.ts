import fs from 'node:fs';
import path from 'node:path';

import { parse } from 'yaml';

import type { FounderDb } from '@/lib/db';
import {
  getCompanyAgentRuntimeProfile,
} from '@/lib/control-plane/agent-runtime-profile';
import {
  StartupMemorySnapshotSchema,
  type StartupMemoryRecord,
  type StartupMemorySnapshot,
} from '@/lib/brain/memory/canonical-schema';

type JsonObject = Record<string, unknown>;

const DEPARTMENT_SPEC_KEYS: Record<string, string> = {
  'dept-research': 'research_investments',
  'dept-risk': 'risk_analysis',
  'dept-monitoring': 'portfolio_monitoring_and_performance_control',
};

function isObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function readYamlFile(rootDir: string, relativePath: string): unknown {
  const absolutePath = path.join(rootDir, relativePath);
  return parse(fs.readFileSync(absolutePath, 'utf8'));
}

function stringify(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function findObjectById(value: unknown, id: string): JsonObject | null {
  if (Array.isArray(value)) {
    for (const item of value) {
      const found = findObjectById(item, id);
      if (found) return found;
    }
    return null;
  }

  if (!isObject(value)) return null;

  if (value.id === id) return value;

  for (const child of Object.values(value)) {
    const found = findObjectById(child, id);
    if (found) return found;
  }

  return null;
}

function collectKeywords(value: unknown, max = 120): string[] {
  const words = new Set<string>();

  const visit = (current: unknown) => {
    if (words.size >= max) return;

    if (typeof current === 'string') {
      for (const token of current
        .toLowerCase()
        .replace(/[_/.-]+/g, ' ')
        .replace(/[^a-z0-9 ]+/g, ' ')
        .split(/\s+/)
        .filter((item) => item.length >= 3)) {
        words.add(token);
        if (words.size >= max) break;
      }
      return;
    }

    if (Array.isArray(current)) {
      current.forEach(visit);
      return;
    }

    if (isObject(current)) {
      for (const [key, child] of Object.entries(current)) {
        visit(key);
        visit(child);
      }
    }
  };

  visit(value);
  return [...words];
}

function record(input: StartupMemoryRecord): StartupMemoryRecord {
  return input;
}

export function buildCanonicalStartupMemory(input: {
  db: FounderDb;
  rootDir?: string;
  generatedAt?: string;
}): StartupMemorySnapshot {
  const rootDir = input.rootDir ?? process.cwd();
  const generatedAt = input.generatedAt ?? new Date().toISOString();

  const companyDoc = readYamlFile(rootDir, 'company/company.yaml');
  const organizationDoc = readYamlFile(rootDir, 'company/organization.yaml');
  const governanceDoc = readYamlFile(rootDir, 'company/governance.yaml');
  const memoryPolicyDoc = readYamlFile(rootDir, 'policies/memory_policy.yaml');

  if (!isObject(companyDoc) || !isObject(companyDoc.company)) {
    throw new Error('invalid_company_specification');
  }

  if (!isObject(organizationDoc) || !isObject(organizationDoc.organization)) {
    throw new Error('invalid_organization_specification');
  }

  if (!isObject(governanceDoc) || !isObject(governanceDoc.governance)) {
    throw new Error('invalid_governance_specification');
  }

  if (!isObject(memoryPolicyDoc) || !isObject(memoryPolicyDoc.startup_brain)) {
    throw new Error('invalid_memory_policy_specification');
  }

  const company = companyDoc.company;
  const organization = organizationDoc.organization;
  const governance = governanceDoc.governance;
  const memoryPolicy = memoryPolicyDoc.startup_brain;

  const records: StartupMemoryRecord[] = [
    record({
      id: 'company:identity',
      kind: 'company',
      title: 'Company identity and operating model',
      summary:
        typeof company.description === 'string'
          ? company.description.trim()
          : 'Canonical company identity, mission and operating model.',
      content: stringify(company),
      keywords: collectKeywords(company),
      version: 1,
      provenance: [
        {
          source: 'company/company.yaml',
          type: 'canonical_yaml',
        },
      ],
      metadata: {
        canonical: true,
      },
    }),


    record({
      id: 'company:memory-policy',
      kind: 'memory_policy',
      title: 'Startup Brain memory policy',
      summary:
        'Canonical policy governing institutional memory, retrieval, learning, provenance and memory lifecycle.',
      content: stringify(memoryPolicy),
      keywords: collectKeywords(memoryPolicy),
      version: 1,
      provenance: [
        {
          source: 'policies/memory_policy.yaml',
          type: 'canonical_yaml',
        },
      ],
      metadata: {
        canonical: true,
      },
    }),
  ];

  for (const [sectionKey, sectionValue] of Object.entries(governance)) {
    if (sectionKey === 'version') continue;

    const sectionContent = {
      version: governance.version ?? '1.0',
      [sectionKey]: sectionValue,
    };

    const extraKeywords =
      sectionKey === 'financial_execution_policy'
        ? ['execute', 'execution', 'trade', 'trades', 'buy', 'sell', 'deposit', 'withdraw', 'order', 'broker']
        : sectionKey === 'separation_of_duties'
          ? ['execute', 'execution', 'trade', 'trades', 'approve', 'approval', 'self', 'permission']
          : [];

    records.push(
      record({
        id: `governance:${sectionKey}`,
        kind: 'governance',
        title: `Governance - ${sectionKey.replace(/_/g, ' ')}`,
        summary: `Canonical governance section: ${sectionKey.replace(/_/g, ' ')}.`,
        content: stringify(sectionContent),
        keywords: [
          ...collectKeywords(sectionContent),
          ...extraKeywords,
        ],
        version: 1,
        provenance: [
          {
            source: `company/governance.yaml#${sectionKey}`,
            type: 'canonical_yaml',
          },
        ],
        metadata: {
          canonical: true,
          section: sectionKey,
        },
      }),
    );
  }

  const departments = [...input.db.departments.all()].sort(
    (a, b) => a.order - b.order,
  );

  for (const department of departments) {
    const specKey = DEPARTMENT_SPEC_KEYS[department.id];
    const departmentSpec =
      specKey && isObject(organization.departments)
        ? organization.departments[specKey]
        : undefined;

    const combined = {
      registry: department,
      organization: departmentSpec ?? null,
    };

    records.push(
      record({
        id: `department:${department.id}`,
        kind: 'department',
        title: department.name,
        summary: department.tagline,
        content: stringify(combined),
        keywords: collectKeywords(combined),
        departmentId: department.id,
        version: 1,
        provenance: [
          {
            source: 'startup.db:departments',
            type: 'startup_db',
          },
          {
            source: 'company/organization.yaml',
            type: 'canonical_yaml',
          },
        ],
        metadata: {
          canonical: true,
          slug: department.slug,
          order: department.order,
        },
      }),
    );
  }

  const agents = [...input.db.agents.all()].sort((a, b) =>
    a.name.localeCompare(b.name),
  );

  for (const agent of agents) {
    const organizationProfile = findObjectById(organization, agent.id);
    const runtimeProfile = getCompanyAgentRuntimeProfile(agent.id);
    const assignmentHistory = input.db.agentModelAssignments.byAgent(agent.id);

    const combined = {
      registry: {
        id: agent.id,
        name: agent.name,
        departmentId: agent.departmentId,
        role: agent.role,
        status: agent.status,
        tier: agent.tier,
        description: agent.description,
        currentModel: agent.model,
        parentId: agent.parentId,
        instance: agent.instance,
      },
      organization: organizationProfile,
      runtimeProfile,
      modelAssignmentHistory: assignmentHistory,
    };

    records.push(
      record({
        id: `agent:${agent.id}`,
        kind: 'agent',
        title: `${agent.name} - ${agent.role}`,
        summary: agent.description,
        content: stringify(combined),
        keywords: [
          agent.id,
          agent.name.toLowerCase(),
          ...collectKeywords(combined),
        ],
        departmentId: agent.departmentId,
        agentId: agent.id,
        currentModel: agent.model,
        version: 1,
        provenance: [
          {
            source: 'startup.db:agents',
            type: 'startup_db',
          },
          {
            source: 'company/organization.yaml',
            type: 'canonical_yaml',
          },
          {
            source: `lib/control-plane/agent-runtime-profile.ts#${agent.id}`,
            type: 'runtime_profile',
          },
          {
            source: `startup.db:agent_model_assignments:${agent.id}`,
            type: 'model_assignment_history',
          },
        ],
        metadata: {
          canonical: true,
          status: agent.status,
          tier: agent.tier,
          parentId: agent.parentId,
          instance: agent.instance,
        },
      }),
    );
  }

  return StartupMemorySnapshotSchema.parse({
    schemaVersion: 1,
    generatedAt,
    records,
  });
}

export const DEFAULT_MODULE_PROFILE = "primitive" as const;

export type ModuleProfile = "primitive";

export interface WorkspaceModuleProfileOption {
  value: ModuleProfile;
  label: string;
  description: string;
}

const PROFILE_OPTIONS: WorkspaceModuleProfileOption[] = [
  {
    value: "primitive",
    label: "Business essentials",
    description: "The foundational operating modules for a new workspace.",
  },
];

export function getWorkspaceModuleProfileOptions(): WorkspaceModuleProfileOption[] {
  return PROFILE_OPTIONS.map((option) => ({ ...option }));
}

export function resolveWorkspaceModuleProfile(
  _settings: Record<string, unknown>,
): ModuleProfile {
  return DEFAULT_MODULE_PROFILE;
}

export function getProfileModuleIds(
  _profile: ModuleProfile,
  _allModuleIds: string[],
  primitiveModuleIds: string[],
): string[] {
  return [...primitiveModuleIds];
}

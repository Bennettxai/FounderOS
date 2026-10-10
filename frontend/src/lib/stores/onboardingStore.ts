import { writable, derived } from "svelte/store";
import { browser } from "$app/environment";
import { getBackendUrl } from "$lib/api/base";

export interface OnboardingWizardState {
  // step index 0-4
  step: number;
  // user display name (step 0)
  displayName: string;
  // org created in step 1
  orgId: string | null;
  orgName: string;
  // workspace created in step 2
  workspaceId: string | null;
  workspaceName: string;
  // completed flag
  completed: boolean;
}

// --- legacy types kept for backward compat with old steps (not actively used) ---

export interface QuickInfo {
  workspaceName: string;
  role: string;
  businessType: string;
  teamSize: string;
}

export interface FallbackFormData {
  toolsUsed: string[];
  mainFocus: string;
  challenge: string;
  workStyle: string;
  whatWouldHelp: string[];
}

export const FALLBACK_OPTIONS = {
  toolsUsed: [
    "Notion",
    "Slack",
    "Linear",
    "Asana",
    "Trello",
    "Monday",
    "Airtable",
    "Jira",
    "HubSpot",
    "Salesforce",
    "Figma",
    "None of these",
  ],
  mainFocus: [
    "Client work & delivery",
    "Building a product",
    "Sales & BD",
    "Operations & processes",
    "Marketing & growth",
    "Creative & design",
  ],
  workStyle: [
    "Deep focus, minimal interruptions",
    "Lots of meetings & collaboration",
    "Async-first",
    "Mix of everything",
  ],
  whatWouldHelp: [
    "Better reporting",
    "Less manual work",
    "Team visibility",
    "Client comms",
    "Project tracking",
    "Time management",
  ],
} as const;

export interface OnboardingState {
  currentStep: number;
  totalSteps: number;
  completed: boolean;
  userData: {
    email?: string;
    username?: string;
    gmailConnected: boolean;
    integrationsConnected: string[];
    interests?: string[];
    starterApps?: StarterApp[];
    quickInfo?: QuickInfo;
    fallbackForm?: FallbackFormData;
  };
  analysis: {
    message1?: string;
    message2?: string;
    message3?: string;
  };
}

export interface StarterApp {
  id: string;
  title: string;
  description: string;
  iconUrl?: string;
  reason: string;
}

const defaultState: OnboardingState = {
  currentStep: 0,
  totalSteps: 5,
  completed: false,
  userData: {
    gmailConnected: false,
    integrationsConnected: [],
  },
  analysis: {},
};

function loadState(): OnboardingState {
  if (browser) {
    try {
      const stored = localStorage.getItem("osa_onboarding_state");
      if (stored) {
        return JSON.parse(stored);
      }
    } catch (e) {
      console.error("Error loading onboarding state:", e);
    }
  }
  return defaultState;
}

function saveState(state: OnboardingState) {
  if (browser) {
    try {
      localStorage.setItem("osa_onboarding_state", JSON.stringify(state));
    } catch (e) {
      console.error("Error saving onboarding state:", e);
    }
  }
}

function createOnboardingStore() {
  const { subscribe, set, update } = writable<OnboardingState>(loadState());

  return {
    subscribe,

    nextStep: () =>
      update((state) => {
        const newState = {
          ...state,
          currentStep: Math.min(state.currentStep + 1, state.totalSteps - 1),
        };
        saveState(newState);
        return newState;
      }),

    prevStep: () =>
      update((state) => {
        const newState = {
          ...state,
          currentStep: Math.max(state.currentStep - 1, 0),
        };
        saveState(newState);
        return newState;
      }),

    goToStep: (step: number) =>
      update((state) => {
        const newState = {
          ...state,
          currentStep: Math.max(0, Math.min(step, state.totalSteps - 1)),
        };
        saveState(newState);
        return newState;
      }),

    setUserData: (data: Partial<OnboardingState["userData"]>) =>
      update((state) => {
        const newState = { ...state, userData: { ...state.userData, ...data } };
        saveState(newState);
        return newState;
      }),

    setAnalysis: (analysis: Partial<OnboardingState["analysis"]>) =>
      update((state) => {
        const newState = {
          ...state,
          analysis: { ...state.analysis, ...analysis },
        };
        saveState(newState);
        return newState;
      }),

    setStarterApps: (apps: StarterApp[]) =>
      update((state) => {
        const newState = {
          ...state,
          userData: { ...state.userData, starterApps: apps },
        };
        saveState(newState);
        return newState;
      }),

    setQuickInfo: (quickInfo: QuickInfo) =>
      update((state) => {
        const newState = {
          ...state,
          userData: { ...state.userData, quickInfo },
        };
        saveState(newState);
        return newState;
      }),

    setFallbackForm: (fallbackForm: FallbackFormData) =>
      update((state) => {
        const newState = {
          ...state,
          userData: { ...state.userData, fallbackForm },
        };
        saveState(newState);
        return newState;
      }),

    setIntegrationsConnected: (integrations: string[]) =>
      update((state) => {
        const newState = {
          ...state,
          userData: { ...state.userData, integrationsConnected: integrations },
        };
        saveState(newState);
        return newState;
      }),

    complete: async () => {
      try {
        const response = await fetch(
          `${getBackendUrl()}/api/users/me/complete-onboarding`,
          {
            method: "POST",
            credentials: "include",
          },
        );
        if (!response.ok) {
          console.error("Failed to mark onboarding complete on backend");
        }
      } catch (error) {
        console.error("Error completing onboarding:", error);
      }

      update((state) => {
        const newState = { ...state, completed: true };
        saveState(newState);
        return newState;
      });
    },

    reset: () => {
      set(defaultState);
      if (browser) {
        localStorage.removeItem("osa_onboarding_state");
      }
    },
  };
}

export const onboardingStore = createOnboardingStore();

export const onboardingProgress = derived(onboardingStore, ($onboarding) =>
  Math.round(($onboarding.currentStep / ($onboarding.totalSteps - 1)) * 100),
);

export const isOnboardingComplete = derived(
  onboardingStore,
  ($onboarding) => $onboarding.completed,
);

// ---------------------------------------------------------------------------
// Wizard store - simple flat state for the new 5-step wizard
// ---------------------------------------------------------------------------

const WIZARD_KEY = "bos_onboarding_wizard";

function defaultWizard(): OnboardingWizardState {
  return {
    step: 0,
    displayName: "",
    orgId: null,
    orgName: "",
    workspaceId: null,
    workspaceName: "",
    completed: false,
  };
}

function loadWizard(): OnboardingWizardState {
  if (!browser) return defaultWizard();
  try {
    const raw = localStorage.getItem(WIZARD_KEY);
    if (raw) return { ...defaultWizard(), ...JSON.parse(raw) };
  } catch {
    // ignore
  }
  return defaultWizard();
}

function saveWizard(s: OnboardingWizardState) {
  if (!browser) return;
  try {
    localStorage.setItem(WIZARD_KEY, JSON.stringify(s));
  } catch {
    // ignore
  }
}

function createWizardStore() {
  const { subscribe, update, set } =
    writable<OnboardingWizardState>(loadWizard());

  function patch(partial: Partial<OnboardingWizardState>) {
    update((s) => {
      const next = { ...s, ...partial };
      saveWizard(next);
      return next;
    });
  }

  return {
    subscribe,
    patch,
    reset: () => {
      const s = defaultWizard();
      set(s);
      if (browser) localStorage.removeItem(WIZARD_KEY);
    },
  };
}

export const wizardStore = createWizardStore();

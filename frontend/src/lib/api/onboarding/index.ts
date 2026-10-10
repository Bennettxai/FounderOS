/**
 * Onboarding API Client
 * Handles communication with onboarding backend endpoints
 */
import { getLegacyApiBaseUrl } from "$lib/config/runtime";

export interface OnboardingSession {
  id: string;
  user_id: string;
  status: "in_progress" | "completed" | "abandoned" | "expired";
  current_step: string;
  steps_completed: string[];
  extracted_data: ExtractedOnboardingData;
  low_confidence_count: number;
  fallback_triggered: boolean;
  workspace_id?: string;
  started_at: string;
  completed_at?: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

export interface ConversationMessage {
  id: string;
  session_id: string;
  role: "user" | "agent" | "system";
  content: string;
  confidence_score?: number;
  extracted_fields?: Record<string, unknown>;
  question_type?: string;
  sequence_number: number;
  created_at: string;
}

export interface ExtractedOnboardingData {
  workspace_name?: string;
  business_type?: string;
  team_size?: string;
  role?: string;
  challenge?: string;
  integrations?: string[];
}

export interface OnboardingStatus {
  needs_onboarding: boolean;
  has_session: boolean;
  session?: OnboardingSession;
  workspace_count: number;
}

export interface SendMessageResponse {
  message: ConversationMessage;
  next_step: string;
  is_complete: boolean;
  should_show_fallback: boolean;
  extracted_data: ExtractedOnboardingData;
  recommended_integrations?: string[];
}

export interface CompleteOnboardingResponse {
  workspace_id: string;
  workspace_name: string;
  workspace_slug: string;
  redirect_url: string;
}

export interface FallbackFormData {
  workspace_name: string;
  business_type: string;
  team_size?: string;
  role?: string;
  challenge?: string;
  integrations?: string[];
}

function getApiBase(): string {
  return getLegacyApiBaseUrl();
}

class OnboardingApiClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
  ): Promise<T> {
    const baseUrl = getApiBase();
    const response = await fetch(`${baseUrl}${endpoint}`, {
      ...options,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response
        .json()
        .catch(() => ({ error: "Request failed" }));
      throw new Error(error.error || error.detail || "Request failed");
    }

    return response.json();
  }

  /** Check if the current user needs onboarding */
  async checkStatus(): Promise<OnboardingStatus> {
    return this.request<OnboardingStatus>("/onboarding/status");
  }

  /**
   * Create a new onboarding session
   */
  async createSession(): Promise<{
    session: OnboardingSession;
    messages: ConversationMessage[];
  }> {
    return this.request("/onboarding/sessions", {
      method: "POST",
    });
  }

  /**
   * Get a session by ID with conversation history
   */
  async getSession(
    sessionId: string,
  ): Promise<{ session: OnboardingSession; messages: ConversationMessage[] }> {
    return this.request(`/onboarding/sessions/${sessionId}`);
  }

  /**
   * Check for and get a resumeable session
   */
  async getResumeableSession(): Promise<{
    has_session: boolean;
    session?: OnboardingSession;
    messages?: ConversationMessage[];
  }> {
    return this.request("/onboarding/resume");
  }

  /**
   * Abandon a session
   */
  async abandonSession(sessionId: string): Promise<{ message: string }> {
    return this.request(`/onboarding/sessions/${sessionId}`, {
      method: "DELETE",
    });
  }

  /** Send a message to the onboarding AI */
  async sendMessage(
    sessionId: string,
    content: string,
  ): Promise<SendMessageResponse> {
    return this.request(`/onboarding/sessions/${sessionId}/messages`, {
      method: "POST",
      body: JSON.stringify({ content }),
    });
  }

  /**
   * Get conversation history for a session
   */
  async getConversationHistory(
    sessionId: string,
  ): Promise<{ messages: ConversationMessage[] }> {
    return this.request(`/onboarding/sessions/${sessionId}/history`);
  }

  /** Select integrations during onboarding */
  async selectIntegrations(
    sessionId: string,
    integrations: string[],
  ): Promise<{ message: string; integrations: string[] }> {
    return this.request(`/onboarding/sessions/${sessionId}/integrations`, {
      method: "POST",
      body: JSON.stringify({ integrations }),
    });
  }

  /**
   * Complete the onboarding and create workspace
   */
  async completeOnboarding(
    sessionId: string,
    integrations: string[] = [],
  ): Promise<CompleteOnboardingResponse> {
    return this.request(`/onboarding/sessions/${sessionId}/complete`, {
      method: "PUT",
      body: JSON.stringify({ integrations }),
    });
  }

  /**
   * Submit fallback form when AI conversation fails
   */
  async submitFallbackForm(
    sessionId: string,
    data: FallbackFormData,
  ): Promise<CompleteOnboardingResponse> {
    return this.request("/onboarding/fallback", {
      method: "POST",
      body: JSON.stringify({ session_id: sessionId, data }),
    });
  }

  /**
   * Get integration recommendations based on session data
   */
  async getRecommendations(sessionId: string): Promise<string[]> {
    const response = await this.request<{ recommendations: string[] }>(
      `/onboarding/sessions/${sessionId}/recommendations`,
    );
    return response.recommendations;
  }
}

// Export singleton instance
export const onboardingApi = new OnboardingApiClient();

// Export class for testing
export { OnboardingApiClient };

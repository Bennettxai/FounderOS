/**
 * Form-related type definitions
 * Provides strong typing for form fields, validation, and form data
 */

/**
 * Configuration for a single form field
 */
export interface FormFieldConfig {
  name: string;
  label: string;
  type:
    | "text"
    | "email"
    | "password"
    | "number"
    | "textarea"
    | "select"
    | "checkbox"
    | "radio";
  required?: boolean;
  placeholder?: string;
  help?: string;
  disabled?: boolean;

  // Validation
  validation?: (value: unknown) => string | null; // Error message or null if valid
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: string; // Regex pattern

  // Select/Radio options
  options?: Array<{ value: string | number; label: string }>;

  // UI hints
  autoComplete?: string;
  rows?: number; // For textarea
  cols?: number; // For textarea
}

/**
 * Configuration for form submission
 */
export interface FormConfig<T> {
  title: string;
  description?: string;
  fields: FormFieldConfig[];
  onSubmit: (data: T) => Promise<void>;
  onCancel?: () => void;
  submitLabel?: string;
  cancelLabel?: string;
  loading?: boolean;
}

/**
 * Errors for specific fields
 */
export interface FieldErrors {
  [fieldName: string]: string;
}

/**
 * General form state tracking
 */
export interface FormState<T> {
  data: T;
  isLoading: boolean;
  isSaving: boolean;
  isEditing: boolean;
  error: string;
  successMessage: string;
  fieldErrors: FieldErrors;
}

/**
 * Agent-specific form data
 * Used for creating/editing agents
 */
export interface AgentFormData {
  id?: string;
  name: string;
  display_name: string;
  description?: string;
  avatar?: string;
  system_prompt: string;
  model_preference?: string;
  temperature?: number;
  max_tokens?: number;
  capabilities?: string[];
  tools_enabled?: string[];
  context_sources?: string[];
  thinking_enabled?: boolean;
  streaming_enabled?: boolean;
  category?: "general" | "specialist" | "system" | "custom";
  is_active?: boolean;
  [key: string]: unknown;
}

/**
 * Command-specific form data
 * Used for creating/editing commands
 */
export interface CommandFormData {
  id?: string;
  name: string;
  display_name: string;
  description?: string;
  icon?: string;
  category?: string;
  system_prompt?: string;
  context_sources?: string[];
  is_custom?: boolean;
  is_builtin_override?: boolean;
  [key: string]: unknown;
}

/**
 * API Key form data
 */
export interface APIKeyFormData {
  provider: string;
  api_key: string;
}

/**
 * Settings form data
 */
export interface SettingsFormData {
  ai_provider: string;
  default_model: string;
  model_settings?: {
    temperature: number;
    maxTokens: number;
    contextWindow: number;
    topP: number;
    streamResponses: boolean;
    showUsageInChat: boolean;
  };
  [key: string]: unknown;
}

/**
 * Validation result
 */
export interface ValidationResult {
  valid: boolean;
  errors: FieldErrors;
}

/**
 * Helper type for form submission handler
 */
export type FormSubmitHandler<T> = (data: T) => Promise<void>;

/**
 * Form response from API
 */
export interface FormResponse<T> {
  ok: boolean;
  data?: T;
  error?: string;
  fieldErrors?: FieldErrors;
}

/**
 * Client-side validation utilities for agent creation/editing forms
 */

import type { CustomAgent } from "$lib/api/ai/types";

export interface ValidationError {
  field: string;
  message: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
}

/**
 * Allowed categories for agents
 */
export const ALLOWED_CATEGORIES = [
  "general",
  "coding",
  "writing",
  "analysis",
  "research",
  "support",
  "sales",
  "marketing",
  "specialist",
  "productivity",
  "creative",
  "technical",
  "custom",
] as const;

/**
 * Validation limits
 */
export const VALIDATION_LIMITS = {
  NAME_MIN: 2,
  NAME_MAX: 50,
  DISPLAY_NAME_MIN: 2,
  DISPLAY_NAME_MAX: 100,
  SYSTEM_PROMPT_MIN: 10,
  SYSTEM_PROMPT_MAX: 5000,
  WELCOME_MESSAGE_MAX: 2000,
  DESCRIPTION_MAX: 500,
  SUGGESTED_PROMPTS_MAX: 10,
  SUGGESTED_PROMPT_MAX: 500,
  TEMPERATURE_MIN: 0.0,
  TEMPERATURE_MAX: 2.0,
  MAX_TOKENS_MIN: 100,
  MAX_TOKENS_MAX: 32000,
} as const;

/**
 * Validate agent form data
 * @param agent Partial agent data to validate
 * @returns ValidationResult with errors if any
 */
export function validateAgentForm(
  agent: Partial<CustomAgent>,
): ValidationResult {
  const errors: ValidationError[] = [];

  // ============ NAME VALIDATION ============
  if (!agent.name || agent.name.trim().length === 0) {
    errors.push({
      field: "name",
      message: "Name is required",
    });
  } else {
    if (agent.name.length < VALIDATION_LIMITS.NAME_MIN) {
      errors.push({
        field: "name",
        message: `Name must be at least ${VALIDATION_LIMITS.NAME_MIN} characters`,
      });
    }
    if (agent.name.length > VALIDATION_LIMITS.NAME_MAX) {
      errors.push({
        field: "name",
        message: `Name cannot exceed ${VALIDATION_LIMITS.NAME_MAX} characters`,
      });
    }
    // Name should be lowercase alphanumeric with hyphens
    if (!/^[a-z0-9-]+$/.test(agent.name)) {
      errors.push({
        field: "name",
        message: "Name must be lowercase alphanumeric with hyphens only",
      });
    }
  }

  // ============ DISPLAY NAME VALIDATION ============
  if (!agent.display_name || agent.display_name.trim().length === 0) {
    errors.push({
      field: "display_name",
      message: "Display name is required",
    });
  } else {
    if (agent.display_name.length < VALIDATION_LIMITS.DISPLAY_NAME_MIN) {
      errors.push({
        field: "display_name",
        message: `Display name must be at least ${VALIDATION_LIMITS.DISPLAY_NAME_MIN} characters`,
      });
    }
    if (agent.display_name.length > VALIDATION_LIMITS.DISPLAY_NAME_MAX) {
      errors.push({
        field: "display_name",
        message: `Display name cannot exceed ${VALIDATION_LIMITS.DISPLAY_NAME_MAX} characters`,
      });
    }
  }

  // ============ SYSTEM PROMPT VALIDATION ============
  if (!agent.system_prompt || agent.system_prompt.trim().length === 0) {
    errors.push({
      field: "system_prompt",
      message: "System prompt is required",
    });
  } else {
    if (agent.system_prompt.length < VALIDATION_LIMITS.SYSTEM_PROMPT_MIN) {
      errors.push({
        field: "system_prompt",
        message: `System prompt must be at least ${VALIDATION_LIMITS.SYSTEM_PROMPT_MIN} characters`,
      });
    }
    if (agent.system_prompt.length > VALIDATION_LIMITS.SYSTEM_PROMPT_MAX) {
      errors.push({
        field: "system_prompt",
        message: `System prompt cannot exceed ${VALIDATION_LIMITS.SYSTEM_PROMPT_MAX} characters`,
      });
    }
  }

  // ============ DESCRIPTION VALIDATION ============
  if (
    agent.description &&
    agent.description.length > VALIDATION_LIMITS.DESCRIPTION_MAX
  ) {
    errors.push({
      field: "description",
      message: `Description cannot exceed ${VALIDATION_LIMITS.DESCRIPTION_MAX} characters`,
    });
  }

  // ============ WELCOME MESSAGE VALIDATION ============
  if (
    agent.welcome_message &&
    agent.welcome_message.length > VALIDATION_LIMITS.WELCOME_MESSAGE_MAX
  ) {
    errors.push({
      field: "welcome_message",
      message: `Welcome message cannot exceed ${VALIDATION_LIMITS.WELCOME_MESSAGE_MAX} characters`,
    });
  }

  // ============ SUGGESTED PROMPTS VALIDATION ============
  if (agent.suggested_prompts) {
    if (
      agent.suggested_prompts.length > VALIDATION_LIMITS.SUGGESTED_PROMPTS_MAX
    ) {
      errors.push({
        field: "suggested_prompts",
        message: `Maximum ${VALIDATION_LIMITS.SUGGESTED_PROMPTS_MAX} suggested prompts allowed`,
      });
    }

    // Validate each prompt
    agent.suggested_prompts.forEach((prompt, index) => {
      if (prompt.trim().length === 0) {
        errors.push({
          field: `suggested_prompt_${index}`,
          message: `Suggested prompt ${index + 1} cannot be empty`,
        });
      }
      if (prompt.length > VALIDATION_LIMITS.SUGGESTED_PROMPT_MAX) {
        errors.push({
          field: `suggested_prompt_${index}`,
          message: `Suggested prompt ${index + 1} cannot exceed ${VALIDATION_LIMITS.SUGGESTED_PROMPT_MAX} characters`,
        });
      }
    });
  }

  // ============ TEMPERATURE VALIDATION ============
  if (agent.temperature !== undefined && agent.temperature !== null) {
    if (
      agent.temperature < VALIDATION_LIMITS.TEMPERATURE_MIN ||
      agent.temperature > VALIDATION_LIMITS.TEMPERATURE_MAX
    ) {
      errors.push({
        field: "temperature",
        message: `Temperature must be between ${VALIDATION_LIMITS.TEMPERATURE_MIN} and ${VALIDATION_LIMITS.TEMPERATURE_MAX}`,
      });
    }
  }

  // ============ MAX TOKENS VALIDATION ============
  if (agent.max_tokens !== undefined && agent.max_tokens !== null) {
    if (
      agent.max_tokens < VALIDATION_LIMITS.MAX_TOKENS_MIN ||
      agent.max_tokens > VALIDATION_LIMITS.MAX_TOKENS_MAX
    ) {
      errors.push({
        field: "max_tokens",
        message: `Max tokens must be between ${VALIDATION_LIMITS.MAX_TOKENS_MIN} and ${VALIDATION_LIMITS.MAX_TOKENS_MAX}`,
      });
    }
  }

  // ============ CATEGORY VALIDATION ============
  if (
    agent.category &&
    !(ALLOWED_CATEGORIES as readonly string[]).includes(agent.category)
  ) {
    errors.push({
      field: "category",
      message: `Category must be one of: ${ALLOWED_CATEGORIES.join(", ")}`,
    });
  }

  // ============ AVATAR URL VALIDATION ============
  if (agent.avatar && agent.avatar.trim().length > 0) {
    try {
      new URL(agent.avatar);
    } catch {
      errors.push({
        field: "avatar",
        message: "Avatar must be a valid URL",
      });
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Get character count status for a field
 * @param current Current character count
 * @param max Maximum allowed characters
 * @returns Object with count info and status class
 */
export function getCharacterCountStatus(current: number, max: number) {
  const percentage = (current / max) * 100;

  return {
    current,
    max,
    percentage,
    remaining: max - current,
    isNearLimit: percentage >= 80,
    isOverLimit: current > max,
    statusClass:
      current > max
        ? "text-red-600 dark:text-red-400 font-semibold"
        : percentage >= 80
          ? "text-orange-600 dark:text-orange-400"
          : "text-gray-500 dark:text-gray-400",
  };
}

/**
 * Validate a single field
 * @param field Field name
 * @param value Field value
 * @returns Error message if invalid, null if valid
 */
export function validateField(
  field: keyof CustomAgent,
  value: unknown,
): string | null {
  const partial: Partial<CustomAgent> = {
    [field]: value as CustomAgent[typeof field],
  };
  const result = validateAgentForm(partial);

  const error = result.errors.find((e) => e.field === field);
  return error ? error.message : null;
}

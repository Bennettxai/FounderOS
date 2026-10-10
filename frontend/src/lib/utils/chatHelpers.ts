/**
 * chatHelpers.ts — Re-exports pure utility functions and types from the chat
 * route's chatActions module so that $lib components can import them via the
 * standard $lib alias rather than crossing the route/lib boundary directly.
 *
 * Usage:
 *   import { renderMarkdown, getArtifactIcon } from '$lib/utils/chatHelpers';
 */
export {
  // Types
  type ModelCapability,
  type ModelOption,
  type ParsedPart,
  // Constant data
  capabilityInfo,
  cloudModelsMap,
  modelContextLimits,
  quickActions,
  greetingSuggestions,
  // Pure functions
  extractThinking,
  renderMarkdown,
  getArtifactIcon,
  getArtifactColor,
  getModelCapabilities,
  getModelDescription,
  parseArtifactsFromContent,
  parseMessageContent,
  formatFileSize,
  formatTime,
  formatTokenCount,
  getContextLimit,
  findBestAssignee,
  getTimeBasedGreeting,
} from "../stores/chat/chatActions";

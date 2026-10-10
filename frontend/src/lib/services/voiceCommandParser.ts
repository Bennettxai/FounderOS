/**
 * Voice Command Parser
 *
 * Orchestrates multi-layer parsing of natural language voice input:
 * wake-word stripping → normalization → pattern matching → conversational routing.
 *
 * Pattern-matching logic lives in voiceCommandPatterns.ts.
 */

import type { VoiceCommand } from "./voiceCommandTypes";
import {
  parseLayoutCommand,
  parseModuleCommand,
  parseResizeCommand,
  parseViewCommand,
  parseNavigationCommand,
} from "./voiceCommandPatterns";

export class VoiceCommandParser {
  /**
   * Detect and strip wake word ("OSA")
   */
  private stripWakeWord(text: string): {
    stripped: string;
    hadWakeWord: boolean;
  } {
    const wakeWords = ["osa", "hey osa", "ok osa"];
    for (const wake of wakeWords) {
      const pattern = new RegExp(`^${wake}[,.]?\\s+`, "i");
      if (pattern.test(text)) {
        return {
          stripped: text.replace(pattern, "").trim(),
          hadWakeWord: true,
        };
      }
    }
    return { stripped: text, hadWakeWord: false };
  }

  /**
   * Extract core command from conversational wrappers
   */
  private extractCommand(text: string): {
    extracted: string;
    confidence: number;
  } {
    const wrappers = [
      {
        pattern: /^(can you|could you|please|can i|i want to)\s+/i,
        confidence: 0.9,
      },
      { pattern: /^(hey|hi|hello),?\s+/i, confidence: 0.8 },
      { pattern: /^(would you|will you)\s+/i, confidence: 0.85 },
    ];

    let extracted = text;
    let confidence = 1.0;

    for (const wrapper of wrappers) {
      if (wrapper.pattern.test(extracted)) {
        extracted = extracted.replace(wrapper.pattern, "");
        confidence *= wrapper.confidence;
      }
    }

    extracted = extracted.replace(/\s+(please|thanks|thank you)[\.\?]?$/i, "");
    return { extracted: extracted.trim(), confidence };
  }

  /**
   * Normalize transcript: remove filler words, fix common transcription errors
   */
  private normalize(text: string): string {
    const fillers = ["um", "uh", "like", "you know", "i mean", "basically"];
    let normalized = text;
    for (const filler of fillers) {
      normalized = normalized.replace(new RegExp(`\\b${filler}\\b`, "gi"), "");
    }

    const corrections: Record<string, string> = {
      "lay out": "layout",
      "auto rotate": "auto-rotate",
      "auto rotation": "auto-rotate",
    };
    for (const [wrong, right] of Object.entries(corrections)) {
      normalized = normalized.replace(new RegExp(wrong, "gi"), right);
    }

    return normalized.trim();
  }

  /**
   * Check if text is conversational (not a command)
   */
  private isConversational(text: string): boolean {
    const conversationalPhrases = [
      "how are you",
      "how are",
      "what are you",
      "tell me about",
      "explain",
      "i need help",
      "i have a question",
      "good morning",
      "good afternoon",
      "good evening",
    ];

    const actionWords = [
      "open",
      "close",
      "show",
      "hide",
      "zoom",
      "expand",
      "contract",
      "switch",
      "go",
      "focus",
      "load",
      "save",
    ];
    if (actionWords.some((word) => text.includes(word))) return false;

    return conversationalPhrases.some((phrase) => text.includes(phrase));
  }

  /**
   * Try all pattern categories in priority order
   */
  private tryAllPatterns(text: string): VoiceCommand | null {
    return (
      parseLayoutCommand(text) ??
      parseModuleCommand(text) ??
      parseResizeCommand(text) ??
      parseViewCommand(text) ??
      parseNavigationCommand(text) ??
      null
    );
  }

  /**
   * Parse a transcript into a structured VoiceCommand
   */
  parse(transcript: string): VoiceCommand {
    const lower = transcript.toLowerCase().trim();

    // Layer 1: Strip wake word
    const { stripped: afterWake } = this.stripWakeWord(lower);

    const normalized = this.normalize(afterWake);

    // Layer 2: Exact pattern matching
    const exactMatch = this.tryAllPatterns(normalized);
    if (exactMatch) {
      return exactMatch;
    }

    // Layer 3: Strip conversational wrapper, retry
    const { extracted, confidence } = this.extractCommand(normalized);

    if (extracted !== normalized && confidence > 0.7) {
      const extractedMatch = this.tryAllPatterns(extracted);
      if (extractedMatch) {
        return extractedMatch;
      }
    }

    // Layer 4: Strict help commands
    const helpPatterns = [
      /^help$/i,
      /^show\s+help$/i,
      /^open\s+help$/i,
      /^display\s+help$/i,
      /^what\s+can\s+i\s+say$/i,
      /^show\s+commands$/i,
      /^list\s+commands$/i,
    ];
    if (helpPatterns.some((p) => p.test(normalized.trim()))) {
      return { type: "help" };
    }

    // Layer 5: Route to AI conversation
    const isQuestion = transcript.includes("?");
    const wordCount = transcript.trim().split(/\s+/).length;
    const isConv = this.isConversational(normalized);

    if (isQuestion || wordCount > 7 || isConv) {
      return { type: "unknown", text: transcript };
    }

    return { type: "unknown", text: transcript };
  }

  /**
   * Get help text for available commands
   */
  getHelpText(): string {
    return `
**Available Voice Commands:**

**Layout Management:**
- "Edit layout" - Enter edit mode
- "Exit edit" - Exit edit mode
- "Save layout as [name]" - Save current layout
- "Load layout [name]" - Switch to a saved layout
- "Manage layouts" - Open layout manager

**Module Navigation:**
- "Open [module]" - Open and focus a module
  Examples: "open chat", "show dashboard", "focus tasks"
- "Close [module]" - Close a module

**View Control:**
- "Switch to orb" / "Switch to grid" - Change view mode
- "Toggle auto-rotate" - Start/stop rotation
- "Zoom in" / "Zoom out" - Adjust zoom level

**Navigation:**
- "Next window" / "Previous window" - Navigate between windows

Say "help" anytime to see this list!
    `.trim();
  }
}

// Export singleton instance
export const voiceCommandParser = new VoiceCommandParser();

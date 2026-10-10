/**
 * Voice Response Utilities
 *
 * Context-aware quick acknowledgment phrases for instant TTS feedback
 * before a full AI response arrives.
 */

type AckMap = Record<string, string[]>;

const ACKS: AckMap = {
  focus_module: [
    "Opening that for you",
    "On it",
    "Let me pull that up",
    "Got it",
  ],
  close_module: ["Closing it down", "Done", "On it", "Sure thing"],
  unfocus: ["Showing all windows", "Back to desktop", "Done", "On it"],
  switch_view: ["Switching views", "Changing it up", "On it", "Here we go"],
  toggle_auto_rotate: ["Got it", "Toggling that", "On it", "Sure"],
  zoom_in: ["Zooming in", "Moving closer", "On it"],
  zoom_out: ["Zooming out", "Moving back", "Got it"],
  reset_zoom: ["Resetting zoom", "Back to normal", "Done"],
  expand_orb: ["Expanding", "Making it bigger", "On it"],
  contract_orb: ["Contracting", "Making it smaller", "Got it"],
  resize_window: ["Resizing", "Adjusting that", "On it"],
  next_window: ["Next one up", "Moving forward", "On it"],
  previous_window: ["Going back", "Previous window", "Got it"],
  enter_edit_mode: ["Entering edit mode", "Let's customize", "On it"],
  exit_edit_mode: ["Exiting edit mode", "Back to normal", "Done"],
  save_layout: ["Saving that layout", "Got it saved", "Done"],
  load_layout: ["Loading that up", "On it", "Switching layouts"],
  default: ["On it", "Got it", "Right away", "Sure thing", "You got it"],
};

function randomFrom(items: string[]): string {
  return items[Math.floor(Math.random() * items.length)];
}

/**
 * Returns a short, randomised acknowledgment string for the given
 * voice command type. Falls back to a generic phrase when the type
 * has no dedicated list.
 */
export function getQuickAck(commandType?: string): string {
  const responses =
    commandType && ACKS[commandType] ? ACKS[commandType] : ACKS.default;
  return randomFrom(responses);
}

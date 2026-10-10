/**
 * Sentence Detection Utilities
 *
 * Smart sentence boundary detection that handles abbreviations.
 * Used by the voice conversation handler to determine when to
 * flush pending TTS text.
 */

export const ABBREVIATIONS: ReadonlySet<string> = new Set([
  "Dr.",
  "Mr.",
  "Mrs.",
  "Ms.",
  "Prof.",
  "Sr.",
  "Jr.",
  "St.",
  "Ave.",
  "Blvd.",
  "Rd.",
  "Ln.",
  "U.S.",
  "U.K.",
  "E.U.",
  "U.N.",
  "etc.",
  "i.e.",
  "e.g.",
  "vs.",
  "approx.",
  "Inc.",
  "Ltd.",
  "Corp.",
  "Co.",
  "Jan.",
  "Feb.",
  "Mar.",
  "Apr.",
  "Jun.",
  "Jul.",
  "Aug.",
  "Sep.",
  "Oct.",
  "Nov.",
  "Dec.",
  "Mon.",
  "Tue.",
  "Wed.",
  "Thu.",
  "Fri.",
  "Sat.",
  "Sun.",
  "a.m.",
  "p.m.",
  "A.M.",
  "P.M.",
]);

/**
 * Returns true when `text` ends at a real sentence boundary.
 * Prevents splitting on common abbreviations (Dr., etc.) or
 * decimal numbers.
 */
export function isCompleteSentence(text: string): boolean {
  const trimmed = text.trim();

  // Known abbreviations — not a sentence end
  for (const abbr of ABBREVIATIONS) {
    if (trimmed.endsWith(abbr)) {
      return false;
    }
  }

  // Single-letter abbreviations: A. B. C.
  if (/\b[A-Z]\.$/.test(trimmed)) {
    return false;
  }

  // Decimal numbers: 3.14, 5.5
  if (/\d+\.\d*$/.test(trimmed)) {
    return false;
  }

  return true;
}

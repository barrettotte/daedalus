// Shared color utilities for palette pickers and hue-to-hex conversion.

export const PALETTE = [
  "#dc2626", "#ea580c", "#ca8a04", "#16a34a", "#0d9488",
  "#2563eb", "#7c3aed", "#c026d3", "#64748b", "#78716c",
];

// Converts an HSL hue (0-360) to a hex color at fixed saturation/lightness.
export function hueToHex(hue: number): string {
  const s = 0.55;
  const l = 0.45;
  const a = s * Math.min(l, 1 - l);

  const f = (n: number) => {
    const k = (n + hue / 30) % 12;
    const c = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
    return Math.round(255 * c).toString(16).padStart(2, "0");
  };

  return `#${f(0)}${f(8)}${f(4)}`;
}

// Validates that a string is a 7-character hex color (e.g. "#1a2b3c").
export function isValidHex(value: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(value);
}

// Computes a hue value (0-360) from a mouse click's x-position on a hue bar element.
export function hueFromClick(e: MouseEvent): number {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
  const raw = Math.round(((e.clientX - rect.left) / rect.width) * 360);
  return Math.max(0, Math.min(360, raw));
}

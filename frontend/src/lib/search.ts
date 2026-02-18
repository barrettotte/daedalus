// Search filtering for board cards. 
// Matches keywords, labels (tag:), and date ranges (created:, due:).

import type { daedalus } from '../../wailsjs/go/models';
import type { BoardLists } from '../stores/board';

// Parsed search token: plain text, #label prefix, #<digits> card ID, @date prefix, url: prefix, or icon: prefix.
export interface SearchToken {
  type: "text" | "label" | "date" | "id" | "url" | "icon";
  value: string;
}

// Parses a query string into typed search tokens.
export function parseSearchTokens(query: string): SearchToken[] {
  const tokens: SearchToken[] = [];
  const parts = query.trim().split(/\s+/);

  for (const part of parts) {
    if (!part) {
      continue;
    }
    if (part.startsWith("#")) {
      const val = part.slice(1);
      if (val && /^\d+$/.test(val)) {
        tokens.push({ type: "id", value: val });
      } else if (val) {
        tokens.push({ type: "label", value: val.toLowerCase() });
      }
    } else if (part.startsWith("@")) {
      const val = part.slice(1);
      if (val) {
        tokens.push({ type: "date", value: val });
      }
    } else if (part.startsWith("url:")) {
      const val = part.slice(4);
      if (val) {
        tokens.push({ type: "url", value: val.toLowerCase() });
      }
    } else if (part.startsWith("icon:")) {
      const val = part.slice(5);
      if (val) {
        tokens.push({ type: "icon", value: val.toLowerCase() });
      }
    } else {
      tokens.push({ type: "text", value: part.toLowerCase() });
    }
  }
  return tokens;
}

// Returns true when a card matches a single search token.
function cardMatchesToken(card: daedalus.KanbanCard, token: SearchToken): boolean {
  if (token.type === "text") {
    const title = (card.metadata.title || "").toLowerCase();
    const preview = (card.previewText || "").toLowerCase();
    const url = (card.metadata.url || "").toLowerCase();
    const checkItems = (card.metadata.checklist || []).map(i => (i.desc || "").toLowerCase());
    return title.includes(token.value)
      || preview.includes(token.value)
      || url.includes(token.value)
      || checkItems.some(desc => desc.includes(token.value));
  }

  if (token.type === "id") {
    return card.metadata.id === Number(token.value);
  }

  if (token.type === "label") {
    const labels = card.metadata.labels || [];
    return labels.some(l => l.toLowerCase().includes(token.value));
  }

  if (token.type === "date") {
    if (!card.metadata.created) {
      return false;
    }
    const created = new Date(card.metadata.created);
    const y = created.getFullYear();
    const m = String(created.getMonth() + 1).padStart(2, "0");
    const d = String(created.getDate()).padStart(2, "0");
    const dateStr = `${y}-${m}-${d}`;
    return dateStr.startsWith(token.value);
  }

  if (token.type === "url") {
    return (card.metadata.url || "").toLowerCase().includes(token.value);
  }

  if (token.type === "icon") {
    return (card.metadata.icon || "").toLowerCase() === token.value;
  }

  return false;
}

// Filters board data by the search query. All tokens must match (AND logic).
export function filterBoard(lists: BoardLists, query: string): BoardLists {
  const tokens = parseSearchTokens(query);
  if (tokens.length === 0) {
    return lists;
  }

  const result: BoardLists = {};
  for (const key of Object.keys(lists)) {
    result[key] = lists[key].filter(card => tokens.every(t => cardMatchesToken(card, t)));
  }
  return result;
}

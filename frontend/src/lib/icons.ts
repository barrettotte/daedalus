// Icon cache module. Caches icon names and content fetched from the backend
// so repeated renders don't trigger RPC calls.

import {
  ListIcons, GetIconContent, SaveCustomIcon as SaveCustomIconRPC,
  DownloadIcon as DownloadIconRPC,
} from "../../wailsjs/go/main/App";

const contentCache: Map<string, string> = new Map();
let iconNamesCache: string[] | null = null;

export async function getIconNames(): Promise<string[]> {
  if (iconNamesCache !== null) {
    return iconNamesCache;
  }
  const names = await ListIcons();
  iconNamesCache = names;
  return names;
}

export async function getIconContent(name: string): Promise<string> {
  const cached = contentCache.get(name);
  if (cached !== undefined) {
    return cached;
  }
  const content = await GetIconContent(name);
  contentCache.set(name, content);
  return content;
}

export async function saveCustomIcon(
  name: string,
  content: string,
): Promise<void> {
  await SaveCustomIconRPC(name, content);
  iconNamesCache = null;
  contentCache.delete(name);
}

export async function downloadIcon(url: string): Promise<string> {
  const filename = await DownloadIconRPC(url);
  iconNamesCache = null;
  contentCache.delete(filename);
  return filename;
}

export function invalidateCache(): void {
  contentCache.clear();
  iconNamesCache = null;
}

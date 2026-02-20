<script lang="ts">
  // Renders a shared SVG icon by name. Uses currentColor for stroke inheritance.

  import { ICON_PATHS } from "../lib/iconPaths";

  let { name, size = 16, style = "" }: { name: string; size?: number; style?: string } = $props();

  let pathData = $derived(ICON_PATHS[name] || '');
</script>

<svg viewBox="0 0 24 24" width={size} height={size} style={style || null}>
  {@html pathData}
</svg>

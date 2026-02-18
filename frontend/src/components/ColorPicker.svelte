<script lang="ts">
  // Shared color picker with palette swatches, custom hue bar, hex input, and reset button.

  import { PALETTE, hueToHex, isValidHex, hueFromClick } from "../lib/color";
  import Icon from "./Icon.svelte";

  let {
    color = $bindable(""),
    onchange,
  }: {
    color: string;
    onchange: (hex: string) => void;
  } = $props();

  let customColorOpen = $state(false);
  let hexInput = $state("");

  // Picks a palette swatch color.
  function pickSwatch(hex: string): void {
    hexInput = hex;
    onchange(hex);
  }

  // Picks a color from the hue bar based on click x-position.
  function handleHueBarClick(e: MouseEvent): void {
    const hex = hueToHex(hueFromClick(e));
    hexInput = hex;
    onchange(hex);
  }

  // Applies the hex input value if it's a valid hex color.
  function applyHexInput(): void {
    const trimmed = hexInput.trim();
    if (isValidHex(trimmed)) {
      onchange(trimmed);
    }
  }

  // Resets (clears) the color.
  function resetColor(): void {
    hexInput = "";
    customColorOpen = false;
    onchange("");
  }
</script>

<div class="color-picker">
  <div class="swatch-row">
    {#each PALETTE as swatch}
      <button class="swatch" class:selected={color === swatch} style="background: {swatch}" title={swatch} onclick={() => pickSwatch(swatch)}></button>
    {/each}
    <button class="custom-toggle" class:active={customColorOpen} onclick={() => { customColorOpen = !customColorOpen; hexInput = color || ''; }} title="Custom color">
      <span class="custom-toggle-rainbow"></span>
      <span class="custom-toggle-plus">+</span>
    </button>
  </div>
  {#if customColorOpen}
    <div class="custom-picker">
      <button type="button" class="hue-bar" title="Pick hue" onclick={handleHueBarClick}></button>
      <div class="hex-row">
        <div class="hex-preview" style="background: {color || 'transparent'}"></div>
        <input type="text" class="form-input hex-input" placeholder="#000000"
          bind:value={hexInput} onblur={applyHexInput}
          onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); applyHexInput(); } }}
        />
      </div>
    </div>
  {/if}
  {#if color}
    <button class="reset-btn" onclick={resetColor}>
      <Icon name="refresh" size={10} />
      Reset color
    </button>
  {/if}
</div>

<style lang="scss">
  .color-picker {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .swatch-row {
    display: flex;
    gap: 5px;
    flex-wrap: wrap;
    align-items: center;
  }

  .swatch {
    all: unset;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    cursor: pointer;
    border: 2px solid transparent;
    box-sizing: border-box;
    transition: border-color 0.15s, transform 0.15s;

    &:hover {
      transform: scale(1.15);
    }

    &.selected {
      border-color: var(--color-text-primary);
    }
  }

  .custom-toggle {
    all: unset;
    position: relative;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    cursor: pointer;
    border: 2px solid transparent;
    box-sizing: border-box;
    transition: border-color 0.15s, transform 0.15s;

    &:hover {
      transform: scale(1.15);
    }

    &.active {
      border-color: var(--color-text-primary);
    }
  }

  .custom-toggle-rainbow {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: conic-gradient(
      #dc2626, #ea580c, #ca8a04, #16a34a, #0d9488,
      #2563eb, #7c3aed, #c026d3, #dc2626
    );
  }

  .custom-toggle-plus {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.65rem;
    font-weight: 700;
    color: #fff;
    text-shadow: 0 0 3px rgba(0, 0, 0, 0.7);
  }

  .custom-picker {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .hue-bar {
    all: unset;
    display: block;
    width: 100%;
    height: 14px;
    border-radius: 4px;
    cursor: crosshair;
    box-sizing: border-box;
    background: linear-gradient(
      to right,
      hsl(0, 55%, 45%),
      hsl(30, 55%, 45%),
      hsl(60, 55%, 45%),
      hsl(90, 55%, 45%),
      hsl(120, 55%, 45%),
      hsl(150, 55%, 45%),
      hsl(180, 55%, 45%),
      hsl(210, 55%, 45%),
      hsl(240, 55%, 45%),
      hsl(270, 55%, 45%),
      hsl(300, 55%, 45%),
      hsl(330, 55%, 45%),
      hsl(360, 55%, 45%)
    );
    border: 1px solid var(--color-border-medium);

    &:hover {
      opacity: 0.9;
    }
  }

  .hex-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .hex-preview {
    width: 20px;
    height: 20px;
    border-radius: 4px;
    border: 1px solid var(--color-border-medium);
    flex-shrink: 0;
  }

  .hex-input {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    min-width: 0;
  }

  .reset-btn {
    all: unset;
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.72rem;
    color: var(--color-text-muted);
    cursor: pointer;
    margin-top: 2px;

    &:hover {
      color: var(--color-text-primary);
    }
  }
</style>

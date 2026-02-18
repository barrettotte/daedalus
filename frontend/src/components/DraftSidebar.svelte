<script lang="ts">
  // Sidebar for the draft card modal. Manages local state via bindable props
  // instead of backend save callbacks -- mirrors CardSidebar's visual structure.

  import { boardData, templates } from "../stores/board";
  import type { daedalus } from "../../wailsjs/go/models";
  import { clickOutside } from "../lib/utils";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";
  import SidebarLabelEditor from "./SidebarLabelEditor.svelte";
  import SidebarIconEditor from "./SidebarIconEditor.svelte";
  import SidebarEstimateEditor from "./SidebarEstimateEditor.svelte";
  import SidebarChecklistSummary from "./SidebarChecklistSummary.svelte";
  import SidebarPositionEditor from "./SidebarPositionEditor.svelte";

  let {
    nextCardId,
    draftListKey = $bindable(""),
    draftPosition = $bindable("top"),
    draftLabels = $bindable<string[]>([]),
    draftIcon = $bindable(""),
    draftDue = $bindable<string | null>(null),
    draftRange = $bindable<{ start: string; end: string } | null>(null),
    draftEstimate = $bindable<number | null>(null),
    draftCounter = $bindable<daedalus.Counter | null>(null),
    draftChecklist = $bindable<daedalus.CheckListItem[] | null>(null),
  }: {
    nextCardId: number;
    draftListKey?: string;
    draftPosition?: string;
    draftLabels?: string[];
    draftIcon?: string;
    draftDue?: string | null;
    draftRange?: { start: string; end: string } | null;
    draftEstimate?: number | null;
    draftCounter?: daedalus.Counter | null;
    draftChecklist?: daedalus.CheckListItem[] | null;
  } = $props();

  let templateDropdownOpen = $state(false);
  let selectedTemplateName = $state("");

  // Cards in the currently selected target list.
  let targetCards = $derived($boardData[draftListKey] || []);

  // Number of position slots -- always N+1 since this is a new card being inserted.
  let positionCount = $derived(targetCards.length + 1);

  // Convert draftPosition string to a 0-based index for the component.
  let positionIndex = $derived.by(() => {
    if (draftPosition === "top") {
      return 0;
    }
    if (draftPosition === "bottom") {
      return positionCount - 1;
    }
    const parsed = parseInt(draftPosition, 10);
    if (!isNaN(parsed) && parsed >= 0 && parsed < positionCount) {
      return parsed;
    }
    return 0;
  });

  function handleSelectPosition(idx: number): void {
    if (idx === 0) {
      draftPosition = "top";
    } else if (idx === positionCount - 1) {
      draftPosition = "bottom";
    } else {
      draftPosition = String(idx);
    }
  }

  // Handles date changes from DateSection.
  function handleDatesChange(due: string | null, range: { start: string; end: string } | null): void {
    draftDue = due;
    draftRange = range;
  }

  // Handles counter changes from CounterControl.
  function handleCounterChange(counter: daedalus.Counter | null): void {
    draftCounter = counter;
  }

  // Resets all template-controlled fields to defaults.
  function resetTemplateFields(): void {
    draftLabels = [];
    draftIcon = "";
    draftEstimate = null;
    draftCounter = null;
    draftChecklist = null;
  }

  // Applies a template's metadata fields to the draft, resetting first.
  function applyTemplate(name: string): void {
    resetTemplateFields();
    selectedTemplateName = name;
    templateDropdownOpen = false;

    const tmpl = $templates.find(t => t.name === name);
    if (!tmpl) {
      return;
    }
    if (tmpl.labels && tmpl.labels.length > 0) {
      draftLabels = [...tmpl.labels];
    }
    if (tmpl.icon) {
      draftIcon = tmpl.icon;
    }
    if (tmpl.estimate != null) {
      draftEstimate = tmpl.estimate;
    }
    if (tmpl.counter) {
      draftCounter = { ...tmpl.counter } as daedalus.Counter;
    }
    if (tmpl.checklist && tmpl.checklist.length > 0) {
      draftChecklist = tmpl.checklist.map((item, i) => ({
        idx: i,
        desc: item.desc,
        done: false,
      })) as daedalus.CheckListItem[];
    }
  }
</script>

<div class="sidebar">
  <div class="sidebar-section">
    <h4 class="sidebar-title">Card #{nextCardId}</h4>
    <SidebarPositionEditor
      listKey={draftListKey}
      position={positionIndex}
      onselectlist={(key) => { draftListKey = key; draftPosition = "top"; }}
      onselectposition={handleSelectPosition}
    />
  </div>
  {#if $templates.length > 0}
    <div class="sidebar-section template-section">
      <span class="template-label">Template</span>
      <div class="tmpl-dropdown" use:clickOutside={() => { templateDropdownOpen = false; }}>
        <button class="tmpl-trigger" onclick={() => { templateDropdownOpen = !templateDropdownOpen; }}>
          <span class="tmpl-trigger-text">{selectedTemplateName || "None"}</span>
          <svg class="tmpl-chevron" class:open={templateDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
            <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
        {#if templateDropdownOpen}
          <div class="tmpl-menu">
            <button class="tmpl-option" class:active={!selectedTemplateName}
              onclick={() => applyTemplate("")}
            >None</button>
            {#each $templates as tmpl}
              <button class="tmpl-option" class:active={tmpl.name === selectedTemplateName}
                onclick={() => applyTemplate(tmpl.name)}
              >{tmpl.name}</button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <SidebarLabelEditor labels={draftLabels || []} onchange={(l) => { draftLabels = l; }} />

  <SidebarIconEditor icon={draftIcon || ""} onchange={(i) => { draftIcon = i; }} />

  <DateSection due={draftDue ?? undefined} range={draftRange ?? undefined} onsave={handleDatesChange} />

  <SidebarEstimateEditor estimate={draftEstimate} onchange={(e) => { draftEstimate = e; }} />

  <CounterControl counter={draftCounter ?? undefined} onsave={handleCounterChange} />

  <SidebarChecklistSummary checklist={draftChecklist} onchange={(c) => { draftChecklist = c; }} />

</div>

<style lang="scss">

  .sidebar-title {
    text-align: center;
  }

  .template-section {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--color-border);
  }

  .template-label {
    font-size: 0.68rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .tmpl-dropdown {
    position: relative;
  }

  .tmpl-trigger {
    all: unset;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 4px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 4px 6px;
    border-radius: 4px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      border-color: var(--color-text-tertiary);
    }
  }

  .tmpl-trigger-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .tmpl-chevron {
    color: var(--color-text-tertiary);
    transition: transform 0.15s;
    flex-shrink: 0;

    &.open {
      transform: rotate(180deg);
    }
  }

  .tmpl-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 0;
    z-index: 10;
    max-height: 200px;
    overflow-y: auto;
  }

  .tmpl-option {
    all: unset;
    display: flex;
    align-items: center;
    width: 100%;
    padding: 5px 8px;
    font-size: 0.8rem;
    color: var(--color-text-primary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
    }

    &.active {
      color: var(--color-accent);
    }
  }
</style>

<script lang="ts">
  // Sidebar for the draft card modal. Manages local state via bindable props
  // instead of backend save callbacks -- mirrors CardSidebar's visual structure.

  import { boardData, templates } from "../stores/board";
  import type { daedalus } from "../../wailsjs/go/models";
  import { clickOutside, autoFocus, blurOnEnter } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";
  import SidebarLabelEditor from "./SidebarLabelEditor.svelte";
  import SidebarIconEditor from "./SidebarIconEditor.svelte";
  import SidebarEstimateEditor from "./SidebarEstimateEditor.svelte";
  import SidebarChecklistSummary from "./SidebarChecklistSummary.svelte";
  import SidebarUriSection from "./SidebarUriSection.svelte";
  import SidebarPositionEditor from "./SidebarPositionEditor.svelte";

  let {
    nextCardId,
    hasUrl = false,
    draftUrl = "",
    onstartedituri,
    onremoveuri,
    editingUri = false,
    draftListKey = "",
    draftPosition = "top",
    onlistkeychange,
    onpositionchange,
    draftLabels = $bindable<string[]>([]),
    draftIcon = $bindable(""),
    draftDue = $bindable<string | null>(null),
    draftRange = $bindable<{ start: string; end: string } | null>(null),
    draftEstimate = $bindable<number | null>(null),
    draftCounter = $bindable<daedalus.Counter | null>(null),
    draftChecklist = $bindable<daedalus.CheckListItem[] | null>(null),
    draftChecklistTitle = $bindable(""),
    draftTimeSeries = $bindable<daedalus.TimeSeries | null>(null),
  }: {
    nextCardId: number;
    hasUrl?: boolean;
    draftUrl?: string;
    onstartedituri?: () => void;
    onremoveuri?: () => void;
    editingUri?: boolean;
    draftListKey?: string;
    draftPosition?: string;
    onlistkeychange: (key: string) => void;
    onpositionchange: (pos: string) => void;
    draftLabels?: string[];
    draftIcon?: string;
    draftDue?: string | null;
    draftRange?: { start: string; end: string } | null;
    draftEstimate?: number | null;
    draftCounter?: daedalus.Counter | null;
    draftChecklist?: daedalus.CheckListItem[] | null;
    draftChecklistTitle?: string;
    draftTimeSeries?: daedalus.TimeSeries | null;
  } = $props();

  let templateDropdownOpen = $state(false);
  let selectedTemplateName = $state("");

  // Time series label editing state.
  let editingTsLabel = $state(false);
  let editTsLabelValue = $state("");

  function startEditTsLabel(): void {
    editTsLabelValue = draftTimeSeries?.label || "Time Series";
    editingTsLabel = true;
  }

  function blurTsLabel(): void {
    editingTsLabel = false;
    const val = editTsLabelValue.trim();
    if (val && draftTimeSeries && val !== (draftTimeSeries.label || "Time Series")) {
      draftTimeSeries = { ...draftTimeSeries, label: val } as daedalus.TimeSeries;
    }
  }

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
      onpositionchange("top");
    } else if (idx === positionCount - 1) {
      onpositionchange("bottom");
    } else {
      onpositionchange(String(idx));
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
    draftChecklistTitle = "";
    draftTimeSeries = null;
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
    if (tmpl.checklist && tmpl.checklist.items && tmpl.checklist.items.length > 0) {
      draftChecklist = tmpl.checklist.items.map((item, i) => ({
        idx: i,
        desc: item.desc,
        done: false,
      })) as daedalus.CheckListItem[];
    }
    if (tmpl.checklist?.label) {
      draftChecklistTitle = tmpl.checklist.label;
    }
    if (tmpl.timeseries && tmpl.timeseries.label) {
      draftTimeSeries = { label: tmpl.timeseries.label, entries: [] } as unknown as daedalus.TimeSeries;
    }
  }
</script>

<div class="sidebar">
  <div class="sidebar-section">
    <h4 class="sidebar-title">Card #{nextCardId}</h4>
    <SidebarPositionEditor
      listKey={draftListKey}
      position={positionIndex}
      onselectlist={(key) => { onlistkeychange(key); onpositionchange("top"); }}
      onselectposition={handleSelectPosition}
    />
  </div>
  {#if $templates.length > 0}
    <div class="sidebar-section template-section">
      <span class="template-label">Template</span>
      <div class="dropdown-wrap" use:clickOutside={() => { templateDropdownOpen = false; }}>
        <button class="dropdown-trigger" onclick={() => { templateDropdownOpen = !templateDropdownOpen; }}>
          <span class="dropdown-trigger-text">{selectedTemplateName || "None"}</span>
          <span class="dropdown-chevron" class:open={templateDropdownOpen}>
            <Icon name="chevron-down" size={12} />
          </span>
        </button>
        {#if templateDropdownOpen}
          <div class="dropdown-menu">
            <button class="dropdown-option" class:active={!selectedTemplateName} onclick={() => applyTemplate("")}>None</button>
            {#each $templates as tmpl}
              <button class="dropdown-option" class:active={tmpl.name === selectedTemplateName} onclick={() => applyTemplate(tmpl.name)}>{tmpl.name}</button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <SidebarLabelEditor labels={draftLabels || []} onchange={(l) => { draftLabels = l; }} />

  <SidebarIconEditor icon={draftIcon || ""} onchange={(i) => { draftIcon = i; }} />

  <SidebarUriSection
    url={hasUrl ? draftUrl : ""}
    editing={editingUri}
    onopen={() => window.open(draftUrl, "_blank")}
    onedit={() => onstartedituri?.()}
    onremove={() => onremoveuri?.()}
  />

  <DateSection due={draftDue ?? undefined} range={draftRange ?? undefined} onsave={handleDatesChange} />

  <SidebarEstimateEditor estimate={draftEstimate} onchange={(e) => { draftEstimate = e; }} />

  <CounterControl counter={draftCounter ?? undefined} onsave={handleCounterChange} />

  <SidebarChecklistSummary checklist={draftChecklist} title={draftChecklistTitle || undefined} onchange={(c) => { draftChecklist = c; }} />

  {#if draftTimeSeries}
    <div class="sidebar-section">
      <div class="section-header">
        {#if editingTsLabel}
          <input class="ts-label-input" type="text" bind:value={editTsLabelValue}
            onclick={(e) => e.stopPropagation()} onblur={blurTsLabel} use:blurOnEnter use:autoFocus
          />
        {:else}
          <h4 class="sidebar-title">{draftTimeSeries.label || "Time Series"}</h4>
        {/if}
        <div class="section-header-actions">
          <button class="counter-header-btn" title="Rename" onclick={startEditTsLabel}>
            <Icon name="pencil" size={12} />
          </button>
          <button class="counter-header-btn remove" title="Remove time series"
            onclick={() => { draftTimeSeries = null; editingTsLabel = false; }}
          >
            <Icon name="trash" size={12} />
          </button>
        </div>
      </div>
    </div>
  {:else}
    <div class="sidebar-section">
      <button class="add-counter-btn" title="Add a time series"
        onclick={() => { draftTimeSeries = { label: "", entries: [] } as unknown as daedalus.TimeSeries; }}
      >
        + Add time series
      </button>
    </div>
  {/if}

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

  .ts-label-input {
    all: unset;
    font-size: 0.75rem;
    color: var(--color-text-secondary);
    border-bottom: 1px solid var(--color-accent);
    padding: 1px 4px;
    min-width: 0;
    flex: 1;
  }
</style>

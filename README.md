# daedalus

Desktop kanban board using directories and markdown files.

## Summary

This is my opinionated way to manage my hobbies, tasks, and time.
Its offline, single user, and single board.

The board is directory/file based to both edit/link in Obsidian and be understood easily by AI agents.
The board is offline, so its up to the user to backup themselves.

I use Obsidian for notetaking, but I keep it as vanilla as possible.
I really don't care for the obsessive customization and installing a bunch of community plugins.
I've used dataviews, bases, and the Kanban plugin, but none of them accomplished exactly what I want.
So this was built specifically as a complementary application to Obsidian.

## Stack

- **Backend**: Go with [Wails](https://wails.io/)
- **Frontend**: Svelte
- **Data**: YAML frontmatter markdown files (directories = lists, `.md` files = cards)

## Card Format

Cards are markdown files with YAML frontmatter:

```yaml
---
title: "Card Title"
id: 123
created: 2026-02-05T06:48:07.044341+00:00
list_order: 48
labels: ["label_1", "label_2"]
due: "2026-03-01"
checklist:
  - { desc: "subtask", done: false }
counter:
  current: 3
  max: 10
  label: "Progress"
range:
  start: 2026-01-01
  end: 2026-06-01
---
# Card Title

Card body...
```

Lists are directories named `##___list_name` (ex: `00___open`, `10___in_progress`).
The prefix controls sort order.

## Development

### Dependencies

```sh
sudo pacman -Syu --needed base-devel gtk3 webkit2gtk svt-av1 libavif npm go
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# verify all wails dependencies installed
wails doctor
```

## Trello Migration

```sh
# Export Trello board as JSON, then
python scripts/trello_to_md.py --input tmp/trello_export.json --output tmp/kanban

# exports:
#  - lists as directories
#  - cards as markdown files with frontmatter yml metadata
```

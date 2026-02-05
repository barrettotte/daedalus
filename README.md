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

## Usage

TODO

## Trello Migration

TODO

## Development

### Dependencies

```sh
sudo pacman -Syu --needed base-devel gtk3 webkit2gtk npm go
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# verify all wails dependencies installed
wails doctor
```

### Useful Commands

```sh
# live
wails dev

# build
wails build
```

## References


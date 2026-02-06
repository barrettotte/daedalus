# convert Trello export JSON to directory/ md file structure
#
# list --> directory
# card --> markdown file

import argparse
from collections import defaultdict
import json
import os
import re

from datetime import datetime, timezone
from pathlib import Path

REPO_DIR = Path(__file__).resolve().parent.parent

DEFAULT_INPUT = REPO_DIR / 'tmp/trello_export.json'
DEFAULT_OUTPUT = REPO_DIR / 'tmp/kanban'

def sanitize_filename(name: str) -> str:
    """Sanitize string for filenames"""
    name = re.sub(r'[<>:"/\\|?*]', '', name) # remove invalid characters
    name = name.replace(' ', '_') # replace spaces with dashes
    name = name.strip(' -') # remove leading/trailing dashes/whitespace
    return name.lower()

def sanitize_link_text(text) -> str:
    """Sanitizes text to be displayed inside a wiki link [[File|Text]]"""
    text = text.replace('[', '(').replace(']', ')') # replace brackets for Obsidian compatibility
    return text.strip()

def get_yaml_checklists(card_id, checklists) -> list:
    """Builds YAML list of tasks"""
    card_checklists = [cl for cl in checklists if cl['idCard'] == card_id]
    yaml_tasks = []

    for cl in card_checklists:
        for item in cl['checkItems']:
            yaml_tasks.append({
                "desc": item['name'],
                "done": item['state'] == 'complete'
            })

    return yaml_tasks

def escape_yaml_string(val: str) -> str:
    """Manually escape string for YAML"""
    if not val:
        return '""'

    val = str(val).replace('\\', '\\\\').replace('"', '\\"')
    val = val.replace('\n', ' ').replace('\r', '')
    return f'"{val}"'

def build_frontmatter(card, labels_map, list_pos, checklists_data) -> str:
    """Build markdown frontmatter for Trello card"""
    lines = ["---"]
    lines.append(f"title: {escape_yaml_string(card['name'])}")
    lines.append(f"id: {int(card['idShort'])}")
    lines.append(f"created: {datetime.now(timezone.utc).isoformat()}")
    lines.append(f"list_order: {list_pos}")

    labels = [labels_map[lbl_id] for lbl_id in card['idLabels'] if lbl_id in labels_map]
    if labels:
        lines.append("labels:")
        for label in labels:
            lines.append(f"  - {escape_yaml_string(label)}")

    if card.get('due'):
        lines.append(f"due: {card['due']}")

    checklists = [cl for cl in checklists_data if cl['idCard'] == card['id']]
    if checklists:
        lines.append("checklist:")
        for cl in checklists:
            for item in sorted(cl['checkItems'], key=lambda x: x['pos']):
                desc = escape_yaml_string(item['name'])
                is_done = "true" if item['state'] == 'complete' else "false"
                lines.append(f"  - {{ desc: {desc}, done: {is_done} }}")

    lines.append("trello_data:")
    lines.append(f"  id: {card['id']}")
    lines.append(f"  url: {card['url']}")
    lines.append(f"  date_closed: {card['dateClosed']}")
    lines.append(f"  date_last_activity: {card['dateLastActivity']}")
    lines.append(f"  date_completed: {card['dateCompleted']}")

    lines.append("---\n")
    return "\n".join(lines)

def main() -> None:
    parser = argparse.ArgumentParser(description="Convert Trello export JSON to directory/markdown file structure")

    parser.add_argument('-i', '--input', type=Path, default=DEFAULT_INPUT,
                        help=f"Path to Trello export JSON (default: {DEFAULT_INPUT})")
    parser.add_argument('-o', '--output', type=Path, default=DEFAULT_OUTPUT,
                        help=f"Output directory for kanban board (default: {DEFAULT_OUTPUT})")

    args = parser.parse_args()

    input_file = args.input
    output_dir = args.output

    print(f"Reading Trello export JSON {input_file}...")
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except FileNotFoundError:
        print(f"Error: Could not find {input_file}.")
        exit(1)

    active_list_ids = set()
    for card in data['cards']:
        active_list_ids.add(card['idList'])

    valid_lists = []
    for lst in sorted(data['lists'], key=lambda k: k['pos']):
        if lst['id'] in active_list_ids:
            valid_lists.append(lst)

    cards_by_list = defaultdict(list)
    for card in data['cards']:
        cards_by_list[card['idList']].append(card)

    labels_map = {l['id']: l['name'] if l['name'] else l['color'] for l in data['labels']}
    folder_idx = 0
    for lst in valid_lists:
        list_id = lst['id']
        list_name = lst['name']

        folder_name = f"{str(folder_idx).zfill(2)}___{sanitize_filename(list_name)}"
        list_dir = os.path.join(output_dir, folder_name)
        os.makedirs(list_dir, exist_ok=True)

        print(f"  Processing List: {list_name}")
        list_cards = cards_by_list.get(list_id, [])
        list_cards.sort(key=lambda k: k['pos'])

        for i, card in enumerate(list_cards, start=1):
            filename = f"{card['idShort']}.md"
            file_path = os.path.join(list_dir, filename)

            body = f"# {card['name']}\n\n{card['desc']}\n"
            frontmatter = build_frontmatter(card, labels_map, i, data.get('checklists', []))

            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(frontmatter)
                f.write(body)

        folder_idx += 1

    print(f"Converted Trello to Markdown at {os.path.abspath(output_dir)}")

if __name__ == "__main__":
    main()

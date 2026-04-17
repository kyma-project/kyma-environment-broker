#!/usr/bin/env python3
"""Updates product-kyma-runtime/docs/toc.yaml when KEB docs structure changes.

Handles:
  - New .md files in docs/: inserts entry into the operations-keb section
  - Renamed .md files: updates the filename reference everywhere in toc.yaml

Only .md additions/renames matter – content changes to existing files are
served automatically via the fileTree reference in manifest.yaml.

Usage:
  update_toc.py --toc <path> --added <file> --renamed <file>

  --added:   path to a file listing added .md paths (one per line,
             relative to repo root, e.g. "docs/contributor/03-85-foo.md")
  --renamed: path to a file listing renames in git --name-status format
             (e.g. "R100\tdocs/old.md\tdocs/new.md")
"""

import sys
import re
import argparse
from pathlib import Path


# ── helpers ───────────────────────────────────────────────────────────────────

# Matches any "- filename: ..." line (with or without surrounding quotes)
FILENAME_RE = re.compile(r'^(\s*-\s+filename:\s+)"?([^"\n]+)"?\s*$')


def parse_numeric_prefix(filename: str):
    """Extract (major, minor) from filenames like '03-85-foo.md'.

    Returns a tuple of ints, or None when the pattern is absent.
    """
    name = Path(filename).name
    m = re.match(r'^(\d+)-(\d+)', name)
    if m:
        return (int(m.group(1)), int(m.group(2)))
    return None


def docs_to_operations_keb(docs_path: str) -> str:
    """Convert 'docs/contributor/foo.md' -> 'operations-keb/contributor/foo.md'."""
    rel = Path(docs_path).relative_to('docs')
    return f"operations-keb/{rel}"


def load_lines(path: str) -> list:
    p = Path(path)
    if not p.exists() or p.stat().st_size == 0:
        return []
    return [line.strip() for line in p.read_text().splitlines() if line.strip()]


# ── text-level operations (no YAML serialisation) ────────────────────────────

def rename_in_lines(lines: list, old_keb: str, new_keb: str) -> tuple:
    """Replace old_keb with new_keb in every filename line.

    Returns (new_lines, changed: bool).
    """
    new_lines = []
    changed = False
    for line in lines:
        m = FILENAME_RE.match(line)
        if m and m.group(2).strip() == old_keb:
            new_lines.append(f'{m.group(1)}"{new_keb}"\n')
            changed = True
        else:
            new_lines.append(line)
    return new_lines, changed


def insert_in_lines(lines: list, new_keb_path: str) -> tuple:
    """Insert a new filename line at the correct position.

    Algorithm:
    1. Determine type: contributor or user.
    2. Among lines referencing operations-keb/ files of the same type,
       find entries whose first number matches the new file's first number.
    3. Among those, pick the one with the largest second number that is
       strictly less than the new file's second number.
    4. Insert right after that line (preserving its indentation).
    5. Fallback: insert after the last operations-keb/ line found.

    Returns (new_lines, inserted: bool).
    """
    new_prefix = parse_numeric_prefix(new_keb_path)
    if new_prefix is None:
        return lines, False

    new_major, new_minor = new_prefix

    if '/contributor/' in new_keb_path:
        file_type = 'contributor'
    elif '/user/' in new_keb_path:
        file_type = 'user'
    else:
        file_type = None

    best_line_idx = -1
    best_minor = -1
    last_keb_line_idx = -1  # fallback anchor

    for i, line in enumerate(lines):
        m = FILENAME_RE.match(line)
        if not m:
            continue
        fname = m.group(2).strip()
        if not fname.startswith('operations-keb/'):
            continue

        last_keb_line_idx = i

        if file_type == 'contributor' and '/contributor/' not in fname:
            continue
        if file_type == 'user' and '/user/' not in fname:
            continue

        prefix = parse_numeric_prefix(fname)
        if prefix is None:
            continue

        entry_major, entry_minor = prefix
        if entry_major != new_major:
            continue

        if entry_minor < new_minor and entry_minor > best_minor:
            best_minor = entry_minor
            best_line_idx = i

    anchor = best_line_idx if best_line_idx >= 0 else last_keb_line_idx

    if anchor < 0:
        print("WARNING: could not find an anchor line – appending at end of file")
        return lines + [f'      - filename: "{new_keb_path}"\n'], True

    # Derive indentation from the anchor line
    anchor_line = lines[anchor]
    indent = len(anchor_line) - len(anchor_line.lstrip())
    new_line = f'{" " * indent}- filename: "{new_keb_path}"\n'

    return lines[:anchor + 1] + [new_line] + lines[anchor + 1:], True


# ── main ──────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Update toc.yaml with KEB doc additions and renames"
    )
    parser.add_argument('--toc', required=True, help='Path to toc.yaml')
    parser.add_argument('--added', required=True,
                        help='File containing list of added .md paths (one per line)')
    parser.add_argument('--renamed', required=True,
                        help='File containing rename entries (git --name-status format)')
    args = parser.parse_args()

    toc_path = Path(args.toc)
    if not toc_path.exists():
        print(f"ERROR: toc.yaml not found at {toc_path}", file=sys.stderr)
        sys.exit(1)

    lines = toc_path.read_text().splitlines(keepends=True)
    changed = False
    warnings = []

    # ── renames ───────────────────────────────────────────────────────────────
    for line in load_lines(args.renamed):
        parts = line.split('\t')
        if len(parts) < 3:
            continue
        old_docs = parts[1].strip()
        new_docs = parts[2].strip()

        if not old_docs.endswith('.md') or not new_docs.endswith('.md'):
            continue
        if 'docs/assets/' in old_docs:
            continue

        old_keb = docs_to_operations_keb(old_docs)
        new_keb = docs_to_operations_keb(new_docs)

        lines, did_change = rename_in_lines(lines, old_keb, new_keb)
        if did_change:
            print(f"Renamed: {old_keb}  →  {new_keb}")
            changed = True
        else:
            warnings.append(
                f"WARNING: '{old_keb}' not found in toc.yaml "
                f"(file may not have been listed yet – skipping rename)"
            )

    # ── additions ─────────────────────────────────────────────────────────────
    all_filenames_in_toc = {
        m.group(2).strip()
        for line in lines
        if (m := FILENAME_RE.match(line))
    }

    for docs_path in load_lines(args.added):
        if not docs_path.endswith('.md'):
            continue
        if 'docs/assets/' in docs_path:
            continue
        if docs_path == 'docs/README.md':
            continue

        keb_path = docs_to_operations_keb(docs_path)

        if keb_path in all_filenames_in_toc:
            print(f"Skipped (already in toc.yaml): {keb_path}")
            continue

        lines, did_insert = insert_in_lines(lines, keb_path)
        if did_insert:
            print(f"Added: {keb_path}")
            all_filenames_in_toc.add(keb_path)
            changed = True

    # ── output ────────────────────────────────────────────────────────────────
    for w in warnings:
        print(w)

    if changed:
        toc_path.write_text(''.join(lines))
        print("toc.yaml updated successfully.")
    else:
        print("No structural changes – toc.yaml left unchanged.")


if __name__ == '__main__':
    main()

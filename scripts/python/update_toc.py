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

# Matches an indented "- filename: name" line in toc.yaml (name optionally in double quotes).
# Group 1 captures the leading indentation + "- filename: " prefix so it can be reused
# when rewriting the line, preserving the original indentation.
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
    return docs_path.replace('docs/', 'operations-keb/', 1)


def load_lines(path: str) -> list:
    p = Path(path)
    if not p.exists() or p.stat().st_size == 0:
        return []
    return [line.strip() for line in p.read_text().splitlines() if line.strip()]


# ── text-level operations (no YAML serialisation) ────────────────────────────

def rename_in_lines(lines: list, old_toc_path: str, new_toc_path: str) -> tuple:
    """Replace old_toc_path with new_toc_path in every matching filename line.

    Returns (new_lines, changed: bool).
    """
    new_lines = []
    changed = False
    for line in lines:
        m = FILENAME_RE.match(line)
        if m and m.group(2).strip() == old_toc_path:
            new_lines.append(line.replace(old_toc_path, new_toc_path))
            changed = True
        else:
            new_lines.append(line)
    return new_lines, changed


def insert_in_lines(lines: list, new_toc_path: str) -> tuple:
    """Insert a new filename line at the correct position.

    Algorithm:
    1. Determine type: contributor or user.
    2. Determine the canonical top-level indentation from the first operations-keb
       entry – only entries at that indentation are considered as anchors, so the
       new line is never inserted inside a subnav block.
    3. Among top-level lines of the same type whose first number matches the new
       file's first number, pick the one with the largest second number strictly
       less than the new file's second number → insert after it.
    4. If no such line exists but the major group is present, insert before the
       first entry of that major group (new file is the smallest in the group).
    5. Fallback: insert after the last top-level operations-keb line found
       (major group doesn't exist yet).

    Returns (new_lines, inserted: bool).
    """
    new_prefix = parse_numeric_prefix(new_toc_path)
    if new_prefix is None:
        return lines, False

    new_major, new_minor = new_prefix

    if '/contributor/' in new_toc_path:
        file_type = 'contributor'
    elif '/user/' in new_toc_path:
        file_type = 'user'
    else:
        file_type = None

    # Determine the canonical indentation for top-level operations-keb entries.
    # Only anchors at this indentation level are considered to avoid inserting
    # inside a subnav block.
    canonical_indent = None
    for line in lines:
        m = FILENAME_RE.match(line)
        if m and m.group(2).strip().startswith('operations-keb/'):
            canonical_indent = len(line) - len(line.lstrip())
            break

    best_line_idx = -1
    best_minor = -1
    first_same_major_idx = -1  # first entry with same major (for insert-before)
    last_keb_line_idx = -1     # fallback when major group doesn't exist yet

    for i, line in enumerate(lines):
        m = FILENAME_RE.match(line)
        if not m:
            continue
        fname = m.group(2).strip()
        if not fname.startswith('operations-keb/'):
            continue

        # Skip subnav children – only anchor to top-level entries
        if canonical_indent is not None and (len(line) - len(line.lstrip())) != canonical_indent:
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

        if first_same_major_idx < 0:
            first_same_major_idx = i

        if entry_minor < new_minor and entry_minor > best_minor:
            best_minor = entry_minor
            best_line_idx = i

    indent = canonical_indent if canonical_indent is not None else 6

    new_line = f'{" " * indent}- filename: "{new_toc_path}"\n'

    if best_line_idx >= 0:
        # Insert after the largest minor that is still smaller than new_minor
        return lines[:best_line_idx + 1] + [new_line] + lines[best_line_idx + 1:], True

    if first_same_major_idx >= 0:
        # New file is the smallest in this major group → insert before the first entry
        return lines[:first_same_major_idx] + [new_line] + lines[first_same_major_idx:], True

    if last_keb_line_idx >= 0:
        # Major group doesn't exist yet → append after the last top-level keb entry
        return lines[:last_keb_line_idx + 1] + [new_line] + lines[last_keb_line_idx + 1:], True

    print("WARNING: could not find an anchor line – appending at end of file")
    return lines + [new_line], True


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
    # Renames are applied in-place: only the filename value is updated.
    # The entry's position in toc.yaml is not changed, as it may be manually
    # curated (e.g. placed inside a subnav or given a specific ordering).
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

        old_toc_path = docs_to_operations_keb(old_docs)
        new_toc_path = docs_to_operations_keb(new_docs)

        lines, did_change = rename_in_lines(lines, old_toc_path, new_toc_path)
        if did_change:
            print(f"Renamed: {old_toc_path}  →  {new_toc_path}")
            changed = True
        else:
            warnings.append(
                f"WARNING: '{old_toc_path}' not found in toc.yaml "
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

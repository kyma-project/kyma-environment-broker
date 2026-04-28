import sys
import os

sys.path.insert(0, os.path.dirname(__file__))

from update_toc import (
    parse_numeric_prefix,
    docs_to_operations_keb,
    rename_in_lines,
    insert_in_lines,
)


# ── fixtures ──────────────────────────────────────────────────────────────────

def make_lines(*entries):
    """Build a list of toc.yaml-style lines from (filename,) or (filename, [children]) tuples."""
    result = []
    for entry in entries:
        if isinstance(entry, str):
            result.append(f'      - filename: "{entry}"\n')
        else:
            parent, children = entry
            result.append(f'      - filename: "{parent}"\n')
            result.append('        subnav:\n')
            for child in children:
                result.append(f'        - filename: "{child}"\n')
    return result


CONTRIBUTOR_LINES = make_lines(
    "operations-keb/README.md",
    "operations-keb/contributor/01-10-authorization.md",
    "operations-keb/contributor/02-30-keb-configuration.md",
    "operations-keb/contributor/03-10-hyperscaler-account-pool.md",
    "operations-keb/contributor/03-40-kyma-bindings-processes.md",
    "operations-keb/contributor/03-50-regions-supporting-machine.md",
    "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
    "operations-keb/contributor/07-10-runtime-reconciler.md",
)

USER_LINES = make_lines(
    "operations-keb/README.md",
    "operations-keb/user/01-10-architecture.md",
    "operations-keb/user/04-10-custom-oidc-configuration.md",
    "operations-keb/user/04-20-custom-administrators.md",
    "operations-keb/user/04-40-additional-worker-node-pools.md",
    "operations-keb/user/04-50-ingress-filtering.md",
    "operations-keb/user/05-10-provisioning-kyma-environment.md",
)

MIXED_LINES = make_lines(
    "operations-keb/README.md",
    "operations-keb/user/01-10-architecture.md",
    "operations-keb/contributor/02-30-keb-configuration.md",
    (
        "operations-keb/contributor/02-40-kyma-template.md",
        [
            "operations-keb/contributor/03-60-regions-configuration.md",
            "operations-keb/contributor/03-55-zones-discovery.md",
        ],
    ),
    "operations-keb/user/04-10-custom-oidc-configuration.md",
    "operations-keb/user/04-40-additional-worker-node-pools.md",
    "operations-keb/contributor/03-50-regions-supporting-machine.md",
    "operations-keb/user/04-50-ingress-filtering.md",
    "operations-keb/contributor/03-10-hyperscaler-account-pool.md",
    "operations-keb/contributor/03-25-assured-workloads.md",
    "operations-keb/user/03-10-service-description.md",
    "operations-keb/user/05-10-provisioning-kyma-environment.md",
    "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
    "operations-keb/contributor/07-10-runtime-reconciler.md",
    "operations-keb/contributor/08-10-cleaning-and-archiving.md",
)


def filenames_from(lines):
    """Extract just the filename values from a list of lines."""
    import re
    pattern = re.compile(r'filename:\s+"?([^"\n]+)"?')
    return [m.group(1).strip() for line in lines if (m := pattern.search(line))]


# ── parse_numeric_prefix ──────────────────────────────────────────────────────

class TestParseNumericPrefix:
    def test_standard(self):
        assert parse_numeric_prefix("03-85-foo.md") == (3, 85)

    def test_leading_zero(self):
        assert parse_numeric_prefix("01-10-architecture.md") == (1, 10)

    def test_from_full_path(self):
        assert parse_numeric_prefix("operations-keb/contributor/06-30-cleanup.md") == (6, 30)

    def test_no_prefix(self):
        assert parse_numeric_prefix("README.md") is None

    def test_single_number_only(self):
        assert parse_numeric_prefix("03-foo.md") is None


# ── docs_to_operations_keb ────────────────────────────────────────────────────

class TestDocsToOperationsKeb:
    def test_contributor(self):
        assert docs_to_operations_keb("docs/contributor/03-85-foo.md") == \
               "operations-keb/contributor/03-85-foo.md"

    def test_user(self):
        assert docs_to_operations_keb("docs/user/01-10-architecture.md") == \
               "operations-keb/user/01-10-architecture.md"

    def test_readme(self):
        assert docs_to_operations_keb("docs/README.md") == "operations-keb/README.md"


# ── rename_in_lines ───────────────────────────────────────────────────────────

class TestRenameInLines:
    def test_renames_existing_entry(self):
        lines = make_lines(
            "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
            "operations-keb/contributor/07-10-runtime-reconciler.md",
        )
        result, changed = rename_in_lines(
            lines,
            "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
            "operations-keb/contributor/06-30-subaccount-cleanup.md",
        )
        assert changed is True
        names = filenames_from(result)
        assert "operations-keb/contributor/06-30-subaccount-cleanup.md" in names
        assert "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md" not in names

    def test_not_found_returns_unchanged(self):
        lines = make_lines("operations-keb/contributor/07-10-runtime-reconciler.md")
        result, changed = rename_in_lines(
            lines,
            "operations-keb/contributor/99-99-does-not-exist.md",
            "operations-keb/contributor/99-99-renamed.md",
        )
        assert changed is False
        assert result == lines

    def test_does_not_touch_other_lines(self):
        lines = make_lines(
            "operations-keb/contributor/01-10-authorization.md",
            "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
            "operations-keb/contributor/08-10-cleaning-and-archiving.md",
        )
        result, _ = rename_in_lines(
            lines,
            "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md",
            "operations-keb/contributor/06-30-subaccount-cleanup.md",
        )
        names = filenames_from(result)
        assert "operations-keb/contributor/01-10-authorization.md" in names
        assert "operations-keb/contributor/08-10-cleaning-and-archiving.md" in names

    def test_renames_entry_in_subnav(self):
        lines = make_lines(
            (
                "operations-keb/contributor/02-40-kyma-template.md",
                ["operations-keb/contributor/03-60-regions-configuration.md"],
            )
        )
        result, changed = rename_in_lines(
            lines,
            "operations-keb/contributor/03-60-regions-configuration.md",
            "operations-keb/contributor/03-60-regions-config.md",
        )
        assert changed is True
        names = filenames_from(result)
        assert "operations-keb/contributor/03-60-regions-config.md" in names
        assert "operations-keb/contributor/03-60-regions-configuration.md" not in names


# ── insert_in_lines ───────────────────────────────────────────────────────────

class TestInsertInLines:
    def test_contributor_inserted_between_neighbors(self):
        # 03-45 should land between 03-40 and 03-50
        result, inserted = insert_in_lines(CONTRIBUTOR_LINES, "operations-keb/contributor/03-45-new.md")
        assert inserted is True
        names = filenames_from(result)
        idx_40 = names.index("operations-keb/contributor/03-40-kyma-bindings-processes.md")
        idx_45 = names.index("operations-keb/contributor/03-45-new.md")
        idx_50 = names.index("operations-keb/contributor/03-50-regions-supporting-machine.md")
        assert idx_40 < idx_45 < idx_50

    def test_user_inserted_between_neighbors(self):
        # 04-45 should land between 04-40 and 04-50
        result, inserted = insert_in_lines(USER_LINES, "operations-keb/user/04-45-new.md")
        assert inserted is True
        names = filenames_from(result)
        idx_40 = names.index("operations-keb/user/04-40-additional-worker-node-pools.md")
        idx_45 = names.index("operations-keb/user/04-45-new.md")
        idx_50 = names.index("operations-keb/user/04-50-ingress-filtering.md")
        assert idx_40 < idx_45 < idx_50

    def test_contributor_appended_when_no_same_major_exists(self):
        # 09-10 – no 09-xx contributor entries exist → fallback: end of list
        result, inserted = insert_in_lines(CONTRIBUTOR_LINES, "operations-keb/contributor/09-10-new-section.md")
        assert inserted is True
        names = filenames_from(result)
        assert names[-1] == "operations-keb/contributor/09-10-new-section.md"

    def test_contributor_inserted_before_smallest_existing_minor_in_same_major(self):
        # 03-05 – only 03-10, 03-40, 03-50 exist (all higher) → insert before 03-10
        result, inserted = insert_in_lines(CONTRIBUTOR_LINES, "operations-keb/contributor/03-05-new.md")
        assert inserted is True
        names = filenames_from(result)
        idx_05 = names.index("operations-keb/contributor/03-05-new.md")
        idx_10 = names.index("operations-keb/contributor/03-10-hyperscaler-account-pool.md")
        assert idx_05 < idx_10

    def test_no_cross_type_interference(self):
        # Adding a contributor file must not anchor against user/ entries.
        # In MIXED_LINES, 03-25 is the last contributor/03-xx with minor < 45,
        # so 03-45 must be inserted right after it.
        result, inserted = insert_in_lines(MIXED_LINES, "operations-keb/contributor/03-45-new.md")
        assert inserted is True
        names = filenames_from(result)
        idx_25 = names.index("operations-keb/contributor/03-25-assured-workloads.md")
        idx_45 = names.index("operations-keb/contributor/03-45-new.md")
        assert idx_25 + 1 == idx_45
        # Must not be anchored to the user/03-10 entry that also has major=3
        idx_user_03 = names.index("operations-keb/user/03-10-service-description.md")
        assert idx_45 != idx_user_03 + 1

    def test_inserted_line_has_double_quotes(self):
        result, _ = insert_in_lines(USER_LINES, "operations-keb/user/04-45-new.md")
        matching = [l for l in result if "04-45-new.md" in l]
        assert len(matching) == 1
        assert '"operations-keb/user/04-45-new.md"' in matching[0]

    def test_inserted_line_preserves_indentation(self):
        result, _ = insert_in_lines(USER_LINES, "operations-keb/user/04-45-new.md")
        anchor = next(l for l in USER_LINES if "04-40" in l)
        new_line = next(l for l in result if "04-45-new.md" in l)
        anchor_indent = len(anchor) - len(anchor.lstrip())
        new_indent = len(new_line) - len(new_line.lstrip())
        assert anchor_indent == new_indent

    def test_contributor_inserted_in_subnav_when_major_group_is_deeper(self):
        # Simulate toc.yaml where top-level entry is README.md and all
        # contributor/06-xx entries live inside its subnav (deeper indentation).
        # 06-80 should be inserted after 06-71 inside the subnav, not after README.md.
        lines = [
            '      - filename: "operations-keb/README.md"\n',
            '        subnav:\n',
            '          - filename: "operations-keb/contributor/06-30-subaccount-cleanup-cronjob.md"\n',
            '          - filename: "operations-keb/contributor/06-71-service-binding-cleanup-cronjob.md"\n',
            '          - filename: "operations-keb/contributor/07-10-runtime-reconciler.md"\n',
        ]
        result, inserted = insert_in_lines(lines, "operations-keb/contributor/06-80-test.md")
        assert inserted is True
        names = filenames_from(result)
        idx_71 = names.index("operations-keb/contributor/06-71-service-binding-cleanup-cronjob.md")
        idx_80 = names.index("operations-keb/contributor/06-80-test.md")
        idx_07 = names.index("operations-keb/contributor/07-10-runtime-reconciler.md")
        assert idx_71 + 1 == idx_80
        assert idx_80 + 1 == idx_07
        # New line must inherit subnav indentation (10 spaces), not top-level (6 spaces)
        new_line = next(l for l in result if "06-80-test.md" in l)
        assert new_line.startswith(' ' * 10)

import os
import sys
import tempfile
from unittest.mock import MagicMock, patch

_module_source = open("scripts/python/resolve_version.py").read()

ENV = {"GITHUB_TOKEN": "test-token", "REPOSITORY": "owner/repo"}
RELEASE_DATE = "2024-01-01T00:00:00Z"
MERGED_DATE = "2024-02-01T00:00:00Z"
OLD_DATE = "2023-12-01T00:00:00Z"


def _make_pr(labels, merged_at=MERGED_DATE, updated_at=None):
    return {
        "merged_at": merged_at,
        "updated_at": updated_at or merged_at or OLD_DATE,
        "labels": [{"name": l} for l in labels],
    }


def _run_detect(latest_version, pr_pages):
    """
    Execute detect_version with mocked API. pr_pages is a list of pages,
    each page being a list of label-lists. An empty terminal page is appended
    automatically to satisfy the pagination loop.
    """
    latest_release = {"name": latest_version, "created_at": RELEASE_DATE}

    built_pages = [
        [_make_pr(labels) for labels in page]
        for page in pr_pages
    ] + [[]]  # terminal empty page

    call_queue = [
        MagicMock(json=lambda lr=latest_release: lr, raise_for_status=lambda: None),
    ] + [
        MagicMock(json=lambda p=page: p, raise_for_status=lambda: None)
        for page in built_pages
    ]

    def fake_get(url, headers=None):
        return call_queue.pop(0)

    with tempfile.NamedTemporaryFile(mode='r', suffix='.env', delete=False) as tmp:
        tmp_path = tmp.name

    try:
        env = {**ENV, "GITHUB_OUTPUT": tmp_path}
        with patch.dict("os.environ", env):
            with patch("requests.get", side_effect=fake_get):
                exec(compile(_module_source, "resolve_version.py", "exec"), {})  # noqa: S102

        with open(tmp_path) as f:
            output = dict(line.strip().split("=", 1) for line in f if "=" in line)
        return output["version"]
    finally:
        os.unlink(tmp_path)


def test_patch_bump_no_feature_prs():
    assert _run_detect("1.2.3", [[["kind/enhancement"], ["kind/bug"]]]) == "1.2.4"


def test_minor_bump_with_feature_pr():
    assert _run_detect("1.2.3", [[["kind/feature"], ["kind/bug"]]]) == "1.3.0"


def test_minor_bump_resets_patch_to_zero():
    assert _run_detect("1.2.9", [[["kind/feature"]]]) == "1.3.0"


def test_patch_bump_no_prs():
    assert _run_detect("1.2.3", []) == "1.2.4"


def test_major_version_preserved():
    assert _run_detect("2.0.0", [[["kind/enhancement"]]]) == "2.0.1"


def test_minor_bump_major_version_preserved():
    assert _run_detect("2.5.3", [[["kind/feature"]]]) == "2.6.0"


def test_v_prefix_stripped():
    assert _run_detect("v1.2.3", [[["kind/enhancement"]]]) == "1.2.4"


def test_v_prefix_with_feature():
    assert _run_detect("v2.3.0", [[["kind/feature"]]]) == "2.4.0"


def test_pagination_feature_on_second_page():
    page1 = [["kind/enhancement"]] * 100
    page2 = [["kind/feature"]]
    assert _run_detect("1.0.0", [page1, page2]) == "1.1.0"


def test_pagination_no_feature_across_pages():
    page1 = [["kind/enhancement"]] * 100
    page2 = [["kind/bug"]] * 50
    assert _run_detect("1.0.0", [page1, page2]) == "1.0.1"


def test_early_exit_on_old_updated_at():
    # Page of 100 PRs all updated before release — should stop without fetching page 2.
    # If page 2 were fetched it would raise IndexError (call_queue empty).
    page1 = [_make_pr(["kind/enhancement"], merged_at=OLD_DATE, updated_at=OLD_DATE)] * 100
    latest_release = {"name": "1.2.3", "created_at": RELEASE_DATE}

    call_queue = [
        MagicMock(json=lambda: latest_release, raise_for_status=lambda: None),
        MagicMock(json=lambda p=page1: p, raise_for_status=lambda: None),
    ]

    def fake_get(url, headers=None):
        return call_queue.pop(0)

    with tempfile.NamedTemporaryFile(mode='r', suffix='.env', delete=False) as tmp:
        tmp_path = tmp.name

    try:
        env = {**ENV, "GITHUB_OUTPUT": tmp_path}
        with patch.dict("os.environ", env):
            with patch("requests.get", side_effect=fake_get):
                exec(compile(_module_source, "resolve_version.py", "exec"), {})  # noqa: S102

        with open(tmp_path) as f:
            output = dict(line.strip().split("=", 1) for line in f if "=" in line)
        assert output["version"] == "1.2.4"
    finally:
        os.unlink(tmp_path)

    latest_release = {"name": "1.2.3", "created_at": RELEASE_DATE}
    old_pr = _make_pr(["kind/feature"], merged_at=OLD_DATE)
    new_pr = _make_pr(["kind/enhancement"], merged_at=MERGED_DATE)

    call_queue = [
        MagicMock(json=lambda: latest_release, raise_for_status=lambda: None),
        MagicMock(json=lambda: [old_pr, new_pr], raise_for_status=lambda: None),
    ]

    def fake_get(url, headers=None):
        return call_queue.pop(0)

    with tempfile.NamedTemporaryFile(mode='r', suffix='.env', delete=False) as tmp:
        tmp_path = tmp.name

    try:
        env = {**ENV, "GITHUB_OUTPUT": tmp_path}
        with patch.dict("os.environ", env):
            with patch("requests.get", side_effect=fake_get):
                exec(compile(_module_source, "resolve_version.py", "exec"), {})  # noqa: S102

        with open(tmp_path) as f:
            output = dict(line.strip().split("=", 1) for line in f if "=" in line)
        assert output["version"] == "1.2.4"
    finally:
        os.unlink(tmp_path)

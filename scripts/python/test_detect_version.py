import importlib.util
import sys
import types
from unittest.mock import MagicMock, patch

# Load detect_version as a module without executing its top-level code
spec = importlib.util.spec_from_file_location("detect_version", "scripts/python/detect_version.py")
_module_source = open("scripts/python/detect_version.py").read()


def _run_detect(latest_version, prs, monkeypatch_env):
    """Execute detect_version logic with mocked API responses and return printed version."""
    latest_release = {"name": latest_version, "created_at": "2024-01-01T00:00:00Z"}
    pr_list = [
        {
            "merged_at": "2024-02-01T00:00:00Z",
            "labels": [{"name": label} for label in labels],
        }
        for labels in prs
    ]

    responses = [
        MagicMock(json=lambda lr=latest_release: lr, raise_for_status=lambda: None),
        MagicMock(json=lambda pl=pr_list: pl, raise_for_status=lambda: None),
    ]

    captured = []

    def fake_get(url, headers=None):
        return responses.pop(0)

    with patch.dict("os.environ", monkeypatch_env):
        with patch("requests.get", side_effect=fake_get):
            with patch("builtins.print", side_effect=lambda *a, **kw: captured.append(str(a[0]))):
                exec(compile(_module_source, "detect_version.py", "exec"), {})  # noqa: S102

    return captured[0]  # first print is the version


ENV = {"GITHUB_TOKEN": "test-token", "REPOSITORY": "owner/repo"}


def test_patch_bump_no_feature_prs():
    version = _run_detect("1.2.3", [["kind/enhancement"], ["kind/bug"]], ENV)
    assert version == "1.2.4"


def test_minor_bump_with_feature_pr():
    version = _run_detect("1.2.3", [["kind/feature"], ["kind/bug"]], ENV)
    assert version == "1.3.0"


def test_minor_bump_resets_patch_to_zero():
    version = _run_detect("1.2.9", [["kind/feature"]], ENV)
    assert version == "1.3.0"


def test_patch_bump_no_prs():
    version = _run_detect("1.2.3", [], ENV)
    assert version == "1.2.4"


def test_major_version_preserved():
    version = _run_detect("2.0.0", [["kind/enhancement"]], ENV)
    assert version == "2.0.1"


def test_minor_bump_major_version_preserved():
    version = _run_detect("2.5.3", [["kind/feature"]], ENV)
    assert version == "2.6.0"

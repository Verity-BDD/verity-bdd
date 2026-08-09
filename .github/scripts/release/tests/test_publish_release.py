import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().parents[1] / "publish_release.py"
spec = importlib.util.spec_from_file_location("publish_release", MODULE_PATH)
assert spec is not None
assert spec.loader is not None
publish_release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publish_release)


SHA = "a" * 40
OTHER_SHA = "b" * 40
REPOSITORY = "Verity-BDD/verity-bdd"
VERSION = "v1.2.3"
BODY = "## Changes\n\n- exact release\n"


def release(**changes: object) -> dict:
    value = {
        "tag_name": VERSION,
        "target_commitish": SHA,
        "name": f"Release {VERSION}",
        "body": BODY,
        "draft": False,
        "prerelease": False,
    }
    value.update(changes)
    return value


def tag_ref(sha: str = SHA, object_type: str = "commit") -> dict:
    return {
        "ref": f"refs/tags/{VERSION}",
        "object": {"type": object_type, "sha": sha},
    }


class FakeClient:
    def __init__(self, responses: list[tuple[str, str, object]]) -> None:
        self.responses = list(responses)
        self.calls: list[tuple[str, str, object | None]] = []

    def request(self, method: str, path: str, payload: object | None = None):
        self.calls.append((method, path, payload))
        if not self.responses:
            raise AssertionError(f"unexpected API call: {method} {path}")
        expected_method, expected_path, response = self.responses.pop(0)
        if (method, path) != (expected_method, expected_path):
            raise AssertionError(
                f"expected {expected_method} {expected_path}, got {method} {path}"
            )
        if isinstance(response, Exception):
            raise response
        return response


def release_path() -> str:
    return f"/repos/{REPOSITORY}/releases/tags/{VERSION}"


def tag_path() -> str:
    return f"/repos/{REPOSITORY}/git/ref/tags/{VERSION}"


def head_path() -> str:
    return f"/repos/{REPOSITORY}/git/ref/heads/main"


class PublishReleaseTest(unittest.TestCase):
    def test_exact_existing_release_and_tag_are_idempotent_success(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (200, tag_ref())),
            ]
        )
        publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertEqual(client.responses, [])
        self.assertFalse(any(method == "POST" for method, _, _ in client.calls))

    def test_conflicting_or_nonfinal_existing_release_fails_closed(self) -> None:
        conflicts = (
            release(target_commitish=OTHER_SHA),
            release(body="different"),
            release(draft=True),
            release(draft=None),
            release(prerelease=True),
            release(prerelease=0),
        )
        for existing in conflicts:
            with self.subTest(existing=existing):
                client = FakeClient([("GET", release_path(), (200, existing))])
                with self.assertRaises(publish_release.PublishError):
                    publish_release.publish(
                        client, REPOSITORY, "main", SHA, VERSION, BODY
                    )

    def test_creates_then_independently_verifies_release_and_tag_at_exact_sha(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (404, {})),
                ("GET", head_path(), (200, {"object": {"type": "commit", "sha": SHA}})),
                ("POST", f"/repos/{REPOSITORY}/releases", (201, release())),
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (200, tag_ref())),
            ]
        )
        publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        post = next(payload for method, _, payload in client.calls if method == "POST")
        self.assertIsInstance(post, dict)
        assert isinstance(post, dict)
        self.assertEqual(post["target_commitish"], SHA)
        self.assertEqual(post["tag_name"], VERSION)
        self.assertEqual(client.responses, [])

    def test_ambiguous_create_failure_reconciles_exact_partial_publication(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (404, {})),
                ("GET", head_path(), (200, {"object": {"type": "commit", "sha": SHA}})),
                ("POST", f"/repos/{REPOSITORY}/releases", publish_release.ApiError("timeout")),
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (200, tag_ref())),
            ]
        )
        publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertEqual(client.responses, [])

    def test_later_retry_creates_missing_release_for_exact_tag_after_main_advanced(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (200, tag_ref())),
                ("POST", f"/repos/{REPOSITORY}/releases", (201, release())),
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (200, tag_ref())),
            ]
        )
        publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertFalse(any(path == head_path() for _, path, _ in client.calls))
        self.assertEqual(client.responses, [])

    def test_later_retry_rejects_conflicting_release_for_exact_existing_tag(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (200, tag_ref())),
                ("POST", f"/repos/{REPOSITORY}/releases", publish_release.ApiError("conflict")),
                ("GET", release_path(), (200, release(body="different"))),
            ]
        )
        with self.assertRaises(publish_release.PublishError):
            publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertFalse(any(path == head_path() for _, path, _ in client.calls))

    def test_stale_main_fails_immediately_before_create(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (404, {})),
                (
                    "GET",
                    head_path(),
                    (200, {"object": {"type": "commit", "sha": OTHER_SHA}}),
                ),
            ]
        )
        with self.assertRaises(publish_release.PublishError):
            publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertFalse(any(method == "POST" for method, _, _ in client.calls))

    def test_conflicting_preexisting_tag_fails_before_create(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (404, {})),
                ("GET", tag_path(), (200, tag_ref(OTHER_SHA))),
            ]
        )
        with self.assertRaises(publish_release.PublishError):
            publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)

    def test_resolves_annotated_tag_to_exact_commit(self) -> None:
        tag_object_sha = "c" * 40
        client = FakeClient(
            [
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (200, tag_ref(tag_object_sha, "tag"))),
                (
                    "GET",
                    f"/repos/{REPOSITORY}/git/tags/{tag_object_sha}",
                    (200, {"object": {"type": "commit", "sha": SHA}}),
                ),
            ]
        )
        publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)
        self.assertEqual(client.responses, [])

    def test_existing_release_without_tag_is_rejected(self) -> None:
        client = FakeClient(
            [
                ("GET", release_path(), (200, release())),
                ("GET", tag_path(), (404, {})),
            ]
        )
        with self.assertRaises(publish_release.PublishError):
            publish_release.publish(client, REPOSITORY, "main", SHA, VERSION, BODY)


if __name__ == "__main__":
    unittest.main()

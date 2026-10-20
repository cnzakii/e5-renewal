import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


GUARD = Path(__file__).with_name("release-guard.sh")


class ReleaseGuardTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        directory = Path(self.temp.name)
        mock = directory / "gh"
        mock.write_text(
            "#!/usr/bin/env python3\n"
            "import os, sys\n"
            "assert sys.argv[1] == 'api'\n"
            "assert sys.argv[2].startswith('repos/owner/tool/releases/tags/v')\n"
            "sys.stdout.write(os.environ['API_RESPONSE'])\n"
            "sys.exit(int(os.environ['API_EXIT']))\n"
        )
        mock.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": str(directory) + os.pathsep + os.environ["PATH"],
            "GITHUB_REPOSITORY": "owner/tool",
        }

    def run_guard(self, tag="v1.2.3", response=None, exit_code=0):
        if response is None:
            response = {"draft": True}
        body = response if isinstance(response, str) else json.dumps(response)
        return subprocess.run(
            ["bash", str(GUARD), tag],
            env={**self.env, "API_RESPONSE": body, "API_EXIT": str(exit_code)},
            text=True,
            capture_output=True,
            check=False,
        )

    def test_draft_can_be_replaced(self):
        result = self.run_guard()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "draft")

    def test_missing_release_can_be_created(self):
        result = self.run_guard(response={"status": "404"}, exit_code=1)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "missing")

    def test_published_release_is_refused(self):
        result = self.run_guard(response={"draft": False})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Refusing to replace published release", result.stderr)

    def test_lookup_failures_do_not_mean_missing(self):
        for response in [{"status": "403"}, {"status": "500"}, "", "invalid"]:
            with self.subTest(response=response):
                result = self.run_guard(response=response, exit_code=1)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, "")

    def test_version_tag_rules(self):
        valid = ["v1.2.3", "v0.0.0", "v1.2.3-rc.1", "v1.2.3-0", "v1.2.3-01a"]
        invalid = [
            "v01.2.3",
            "v1.02.3",
            "v1.2.03",
            "v1.2.3-01",
            "v1.2.3-rc.01",
            "v1.2.3-rc..1",
            "v1.2.3-",
            "v1.2.3+build.1",
            "vlatest",
            "v1.2",
            "v1.2.3/evil",
        ]
        for tag in valid + invalid:
            with self.subTest(tag=tag):
                result = self.run_guard(tag=tag)
                self.assertEqual(result.returncode == 0, tag in valid, result.stderr)


if __name__ == "__main__":
    unittest.main()

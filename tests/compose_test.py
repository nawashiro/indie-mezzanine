"""Composeの回帰検証。実トークンとDockerデーモンは使わない。"""
import json
import os
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]


def config(token="compose-test-token", data=None):
    env = os.environ.copy()
    for key in ("CLOUDFLARED", "HOST_DATA_DIR", "PUBLIC_URL", "PORT"):
        env.pop(key, None)
    env["PUBLIC_URL"] = "https://relay.example/"
    if token is not None:
        env["CLOUDFLARED"] = token
    if data is not None:
        env["HOST_DATA_DIR"] = data
    return subprocess.run(
        ["docker", "compose", "--env-file", "/dev/null", "-f",
         str(ROOT / "compose.yml"), "config", "--format", "json"],
        cwd=ROOT, env=env, capture_output=True, text=True,
    )


class ComposeTests(unittest.TestCase):
    def test_private_network_and_bind_mount(self):
        result = config()
        self.assertEqual(result.returncode, 0, "Compose設定検証失敗")
        doc = json.loads(result.stdout)
        self.assertEqual(set(doc["services"]), {"relay", "cloudflared"})
        self.assertFalse(doc.get("volumes"))
        for service in doc["services"].values():
            self.assertFalse(service.get("ports"))
            self.assertNotEqual(service.get("network_mode"), "host")
            self.assertEqual(service["user"], "65532:65532")
            self.assertTrue(service["read_only"])
            self.assertIn("ALL", service["cap_drop"])
            self.assertGreater(int(service["mem_limit"]), 0)
            self.assertGreater(float(service["cpus"]), 0)
            for volume in service.get("volumes", []):
                self.assertNotIn("docker.sock", volume["target"])
        relay = doc["services"]["relay"]
        tunnel = doc["services"]["cloudflared"]
        self.assertTrue(set(relay["networks"]) & set(tunnel["networks"]))
        self.assertFalse(any(n.get("internal") for n in doc["networks"].values()))
        volume, = relay["volumes"]
        self.assertEqual(volume["type"], "bind")
        self.assertEqual(volume["source"], str(ROOT / "data"))
        self.assertEqual(volume["target"], "/data")
        self.assertFalse(volume["bind"]["create_host_path"])
        self.assertEqual(relay["environment"]["DATA_DIR"], "/data")
        self.assertRegex(tunnel["image"], r":\d{4}\.\d+\.\d+@sha256:[a-f0-9]{64}$")
        self.assertEqual(tunnel["depends_on"]["relay"]["condition"], "service_healthy")

    def test_token_is_only_in_tunnel_environment(self):
        result = config()
        self.assertEqual(result.returncode, 0, "Compose設定検証失敗")
        doc = json.loads(result.stdout)
        self.assertEqual(doc["services"]["cloudflared"]["environment"],
                         {"TUNNEL_TOKEN": "compose-test-token"})
        self.assertNotIn("compose-test-token", json.dumps(doc["services"]["relay"]))
        self.assertEqual(doc["services"]["cloudflared"]["command"],
                         ["tunnel", "--no-autoupdate", "run"])

    def test_missing_or_empty_token_is_rejected(self):
        for token in (None, ""):
            with self.subTest(token=token):
                result = config(token)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("CLOUDFLARED", result.stderr)
                self.assertNotIn("compose-test-token", result.stderr)

    def test_custom_host_data_directory(self):
        result = config(data="/tmp/mezzanine-compose-test-data")
        self.assertEqual(result.returncode, 0, "Compose設定検証失敗")
        volume, = json.loads(result.stdout)["services"]["relay"]["volumes"]
        self.assertEqual(volume["source"], "/tmp/mezzanine-compose-test-data")
        self.assertEqual(volume["target"], "/data")


if __name__ == "__main__":
    unittest.main()

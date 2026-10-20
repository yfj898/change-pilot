"""Sync the user's existing SDK, preserving a standalone WeKnora Go module."""
import argparse
from pathlib import Path

FILES = ("client.go", "client_test.go", "operations.go", "operations_test.go", "verification.go", "verification_test.go")
parser = argparse.ArgumentParser()
parser.add_argument("source", type=Path, help="Explicit path to TaskFlow pkg/cloudclient")
args = parser.parse_args()
source = args.source.resolve(strict=True)
destination = Path(__file__).resolve().parents[1] / "internal/integration/taskflowclient"
payload = {name: (source / name).read_bytes() for name in FILES}
if any(b"package cloudclient" not in data for data in payload.values()):
    raise SystemExit("source must be the existing TaskFlow cloudclient package")
destination.mkdir(parents=True, exist_ok=True)
for name, data in payload.items():
    (destination / name).write_bytes(data)
for name, data in payload.items():
    if (destination / name).read_bytes() != data:
        raise SystemExit(f"copy verification failed: {name}")
print(f"Copied and byte-compared {len(FILES)} SDK source/test files from {source}")

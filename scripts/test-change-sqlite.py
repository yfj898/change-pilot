"""Exercise the exact additive SQLite schema without native Go toolchain dependencies."""
from pathlib import Path
import sqlite3
import unittest

ROOT = Path(__file__).resolve().parents[1]
UP = ROOT / "migrations/sqlite/000042_change_workspace.up.sql"
DOWN = ROOT / "migrations/sqlite/000042_change_workspace.down.sql"


class ChangeSchemaTest(unittest.TestCase):
    def test_additive_schema_and_constraints(self):
        with sqlite3.connect(":memory:") as db:
            db.execute("PRAGMA foreign_keys=ON")
            # A preexisting unrelated table must survive both directions.
            db.execute("CREATE TABLE original_resource(id TEXT PRIMARY KEY)")
            db.execute("INSERT INTO original_resource VALUES('retained')")
            db.executescript(UP.read_text(encoding="utf-8"))
            db.executescript((ROOT/"migrations/sqlite/000043_change_runs.up.sql").read_text(encoding="utf-8"))
            db.executescript((ROOT/"migrations/sqlite/000044_change_draft.up.sql").read_text(encoding="utf-8"))
            db.execute("""INSERT INTO change_requests
                (id,tenant_id,creator_id,title,input,revision,input_revision,stage,created_at,updated_at)
                VALUES('change',1,'creator','Title','{}',1,1,'draft','2026-10-19','2026-10-19')""")
            db.execute("""INSERT INTO change_snapshots
                VALUES('snapshot',1,'change',1,1,'{}','creator','2026-10-19')""")
            with self.assertRaises(sqlite3.IntegrityError):
                db.execute("INSERT INTO change_snapshots VALUES('duplicate',1,'change',1,1,'{}','creator','2026-10-19')")
            with self.assertRaises(sqlite3.IntegrityError):
                db.execute("UPDATE change_requests SET stage='passed' WHERE id='change'")
            db.executescript((ROOT/"migrations/sqlite/000044_change_draft.down.sql").read_text(encoding="utf-8"))
            db.executescript((ROOT/"migrations/sqlite/000043_change_runs.down.sql").read_text(encoding="utf-8"))
            db.executescript(DOWN.read_text(encoding="utf-8"))
            self.assertEqual(db.execute("SELECT id FROM original_resource").fetchall(), [("retained",)])


if __name__ == "__main__":
    unittest.main()

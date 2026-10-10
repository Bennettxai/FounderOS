defmodule OptimalEngine.BackupTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Backup

  @moduletag :tmp_dir

  test "create/1 produces a readable SQLite file and counts rows", %{tmp_dir: tmp} do
    target = Path.join(tmp, "snap.db")

    assert {:ok, info} = Backup.create(target)
    assert info.target == target
    assert info.size_bytes > 0
    assert info.rows_backed_up >= 0
    assert info.duration_ms >= 0
    assert String.starts_with?(info.id, "backup_")
    assert String.length(info.checksum_sha256) == 64
    assert info.integrity_status == "verified"

    assert File.exists?(target)

    assert {:ok, [["completed", "verified", checksum]]} =
             OptimalEngine.Store.raw_query(
               "SELECT status, integrity_status, checksum_sha256 FROM backup_records WHERE id = ?1",
               [info.id]
             )

    assert checksum == info.checksum_sha256
  end

  test "create/1 refuses to overwrite an existing target", %{tmp_dir: tmp} do
    target = Path.join(tmp, "existing.db")
    File.write!(target, "")
    assert {:error, {:target_exists, ^target}} = Backup.create(target)
  end

  test "create/1 rejects a non-existent parent directory", %{tmp_dir: tmp} do
    target = Path.join([tmp, "nope", "snap.db"])
    assert {:error, {:parent_missing, _}} = Backup.create(target)
  end

  test "verify/1 returns :ok on a clean backup", %{tmp_dir: tmp} do
    target = Path.join(tmp, "verify.db")
    {:ok, _} = Backup.create(target)
    assert {:ok, :ok} = Backup.verify(target)
  end

  test "restore/1 refuses while the supervisor is up", %{tmp_dir: tmp} do
    target = Path.join(tmp, "snap.db")
    {:ok, _} = Backup.create(target)
    assert {:error, :supervisor_still_running} = Backup.restore(target)
  end
end

# Skip RocksDB backend tests unless the NIF is loadable
exclude = if Code.ensure_loaded?(:rocksdb), do: [], else: [:rocksdb]
ExUnit.configure(exclude: exclude)

ExUnit.start()

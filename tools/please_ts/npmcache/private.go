package npmcache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Private makes denoDir a DENO_DIR that reads the shared Deno cache at shared but writes
// only to itself.
//
// Deno writes to its DENO_DIR on every run, even with everything cached: the SQLite databases
// it keeps next to the npm packages (the type check, the dependency analysis, the V8 code
// cache) and, for a script it transpiles, the transpiled file. Targets run in parallel, so
// running them all on one shared directory has many processes writing to the same SQLite
// files at once, and a build that copies the directory meanwhile sees it change under it.
//
// The npm packages are the bulk of the cache (tens of MB) and are only read when they are
// cached already, so they are linked. The databases are small (a few MB), so each run gets its
// own copy, without the -wal and -shm files that belong to a connection of the shared one.
// What a run learns stays in its directory and is dropped with it.
func Private(shared, denoDir string) error {
	if err := os.MkdirAll(denoDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(shared)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := adopt(shared, denoDir, e); err != nil {
			return fmt.Errorf("making a private copy of the Deno cache: %w", err)
		}
	}
	return nil
}

// adopt puts one entry of the shared cache into the private directory: the npm packages
// linked, a database copied, anything else (such as the -wal and -shm files) left out.
func adopt(shared, denoDir string, e os.DirEntry) error {
	src := filepath.Join(shared, e.Name())
	dst := filepath.Join(denoDir, e.Name())
	switch {
	case e.IsDir() && e.Name() == "npm":
		abs, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		return os.Symlink(abs, dst)
	case e.Type().IsRegular() && !isConnectionFile(e.Name()):
		return copyFile(src, dst)
	}
	return nil
}

// isConnectionFile is whether a file is the write-ahead log or the shared memory of an open
// SQLite database, which belong to the process that has it open.
func isConnectionFile(name string) bool {
	return strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm")
}

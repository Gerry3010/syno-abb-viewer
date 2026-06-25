package remotefs

// DirSize returns the recursive total byte size and file count under dir.
// Unreadable subdirectories are skipped rather than aborting the whole walk —
// a single permission error deep in a backup tree shouldn't blank out a run's size.
func DirSize(fsys FS, dir string) (size int64, files int, err error) {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if e.IsDir {
			s, f, derr := DirSize(fsys, e.Path)
			if derr != nil {
				continue // skip subtree we can't read
			}
			size += s
			files += f
			continue
		}
		size += e.Size
		files++
	}
	return size, files, nil
}

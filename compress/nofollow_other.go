//go:build windows

package compress

// oNoFollow is 0 on platforms that lack O_NOFOLLOW (e.g. Windows). The explicit
// os.Lstat check in extractTarFile still refuses to write through a pre-existing
// symlink there.
const oNoFollow = 0

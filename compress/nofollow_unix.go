//go:build !windows

package compress

import "syscall"

// oNoFollow makes OpenFile refuse to traverse a symlink at the final path
// element (defense against symlink TOCTOU when writing extracted files). It is
// only defined on platforms that support O_NOFOLLOW; see nofollow_other.go for
// the fallback.
const oNoFollow = syscall.O_NOFOLLOW

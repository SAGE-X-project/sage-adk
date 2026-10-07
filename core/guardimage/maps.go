// SPDX-License-Identifier: LGPL-3.0-or-later
package guardimage

import (
	"bufio"
	"bytes"
	"runtime"
	"strconv"
	"strings"
)

func validMaps(raw []byte, majorDevice, minorDevice, inode uint64) bool {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	found := false
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || len(fields[1]) != 4 {
			return false
		}
		permissions := fields[1]
		if (permissions[0] != 'r' && permissions[0] != '-') ||
			(permissions[1] != 'w' && permissions[1] != '-') ||
			(permissions[2] != 'x' && permissions[2] != '-') ||
			(permissions[3] != 'p' && permissions[3] != 's') {
			return false
		}
		if permissions[2] != 'x' {
			continue
		}
		if permissions[1] == 'w' {
			return false
		}
		dev := strings.Split(fields[3], ":")
		if len(dev) != 2 {
			return false
		}
		major, e1 := strconv.ParseUint(dev[0], 16, 32)
		minor, e2 := strconv.ParseUint(dev[1], 16, 32)
		ino, e3 := strconv.ParseUint(fields[4], 10, 64)
		if e1 != nil || e2 != nil || e3 != nil {
			return false
		}
		if ino == 0 {
			if major != 0 || minor != 0 || len(fields) != 6 ||
				(fields[5] != "[vdso]" && (runtime.GOARCH != "amd64" || fields[5] != "[vsyscall]")) {
				return false
			}
			continue
		}
		if major != majorDevice || minor != minorDevice || ino != inode {
			return false
		}
		found = true
	}
	return found && scanner.Err() == nil
}

//go:build !embedui

package frontendassets

import "io/fs"

func StaticFS() (fs.FS, bool, error) {
	return nil, false, nil
}

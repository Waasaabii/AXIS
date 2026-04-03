//go:build embedui

package frontendassets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedDist embed.FS

func StaticFS() (fs.FS, bool, error) {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, false, err
	}
	return sub, true, nil
}

// Package catalog embeds the official service catalog into the one binary.
//
// The directory tree under services/ is compiled into the binary at build
// time via //go:embed and exposed as an fs.FS rooted at the services level
// (so consumers see <service>/service.yaml, not services/<service>/...).
package catalog

import (
	"embed"
	"io/fs"
)

//go:embed services
var embedded embed.FS

// Services returns the embedded official catalog rooted at the services
// directory. Each entry is a service directory.
func Services() fs.FS {
	sub, err := fs.Sub(embedded, "services")
	if err != nil {
		// fs.Sub on an embedded path only fails on programmer error
		// (invalid path); panic so the bug surfaces at startup.
		panic("catalog embed: " + err.Error())
	}
	return sub
}

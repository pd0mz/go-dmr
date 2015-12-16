package dmr

import (
	"fmt"
	"runtime"
)

var (
	Version    = "0.2.1"                                            // Version number
	SoftwareID = fmt.Sprintf("%s go-dmr %s", Version, runtime.GOOS) // Software identifier
	PackageID  = fmt.Sprintf("%s/%s", SoftwareID, runtime.GOARCH)   // Package identifier
)

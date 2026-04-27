package version

// Version is the gmr release version. It can be overridden at build time via:
//
//	go build -ldflags "-X github.com/slucheninov/gmr/internal/version.Version=vX.Y.Z"
var Version = "0.6.1"

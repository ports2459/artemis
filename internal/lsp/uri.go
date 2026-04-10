package lsp

import (
	"net/url"
	"runtime"
	"strings"
)

// FileToURI converts a filesystem path to a file:// URI.
// Example: /home/user/file.cs -> file:///home/user/file.cs
func FileToURI(path string) string {
	// On Windows, paths start with a drive letter like C:\
	// and need an extra slash: file:///C:/...
	if runtime.GOOS == "windows" {
		path = "/" + strings.ReplaceAll(path, `\`, "/")
	}
	u := &url.URL{
		Scheme: "file",
		Path:   path,
	}
	return u.String()
}

// URIToFile converts a file:// URI to a filesystem path.
// Example: file:///home/user/file.cs -> /home/user/file.cs
func URIToFile(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		// Best-effort fallback: strip the scheme prefix.
		after, ok := strings.CutPrefix(uri, "file://")
		if ok {
			return after
		}
		return uri
	}
	path := u.Path
	// On Windows, strip the leading slash from /C:/...
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
		path = strings.ReplaceAll(path, "/", `\`)
	}
	return path
}

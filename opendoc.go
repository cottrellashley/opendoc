// Package opendoc provides the embedded assets for the OpenDoc static site generator.
package opendoc

import "embed"

// ThemesFS contains all built-in theme files (templates, CSS, JS).
//
//go:embed all:themes
var ThemesFS embed.FS

// PublicFS contains the workbench UI files (HTML, CSS, JS).
//
//go:embed all:public
var PublicFS embed.FS

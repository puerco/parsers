// SPDX-License-Identifier: Apache-2.0

package gomod

import (
	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type GoMod struct {
	metadata   plugin.Metadata
	rootModule *meta.Package
	command    *helper.Cmd
}

type JSONOutput struct {
	Dir        string  `json:"Dir,omitempty"`
	ImportPath string  `json:"ImportPath,omitempty"`
	Name       string  `json:"Name,omitempty"`
	Module     *Module `json:"Module,omitempty"`
}

type Module struct {
	Version   string     `json:"Version,omitempty"`
	Path      string     `json:"Path,omitempty"`
	Dir       string     `json:"Dir,omitempty"`
	Replace   modReplace `json:"Replace,omitempty"`
	GoMod     string     `json:"GoMod,omitempty"`
	GoVersion string     `json:"GoVersion,omitempty"`
}

type modReplace struct {
	Path      string `json:"Path,omitempty"`
	Dir       string `json:"Dir,omitempty"`
	GoMod     string `json:"GoMod,omitempty"`
	GoVersion string `json:"GoVersion,omitempty"`
}

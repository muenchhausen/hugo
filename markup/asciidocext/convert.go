// Copyright 2020 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package asciidocext converts Asciidoc to HTML using Asciidoc or Asciidoctor
// external binaries. The `asciidoc` module is reserved for a future golang
// implementation.
package asciidocext

import (
	"os/exec"
	"path/filepath"

	"github.com/gohugoio/hugo/identity"
	"github.com/gohugoio/hugo/markup/asciidocext/asciidocext_config"
	"github.com/gohugoio/hugo/markup/converter"
	"github.com/gohugoio/hugo/markup/internal"
)

/* ToDo: RelPermalink patch for svg posts not working
type pageSubset interface {
	RelPermalink() string
}
*/

// Provider is the package entry point.
var Provider converter.ProviderProvider = provider{}

type provider struct{}

func (p provider) New(cfg converter.ProviderConfig) (converter.Provider, error) {
	return converter.NewProvider("asciidocext", func(ctx converter.DocumentContext) (converter.Converter, error) {
		return &asciidocConverter{
			ctx: ctx,
			cfg: cfg,
		}, nil
	}), nil
}

type asciidocConverter struct {
	ctx converter.DocumentContext
	cfg converter.ProviderConfig
}

func (a *asciidocConverter) Convert(ctx converter.RenderContext) (converter.Result, error) {
	return converter.Bytes(a.getAsciidocContent(ctx.Src, a.ctx)), nil
}

func (c *asciidocConverter) Supports(feature identity.Identity) bool {
	return false
}

// getAsciidocContent calls asciidoctor or asciidoc as an external helper
// to convert AsciiDoc content to HTML.
func (a *asciidocConverter) getAsciidocContent(src []byte, ctx converter.DocumentContext) []byte {
	path := getAsciidoctorExecPath()
	if path == "" {
		a.cfg.Logger.ERROR.Println("asciidoctor / asciidoc not found in $PATH: Please install.\n",
			"                 Leaving AsciiDoc content unrendered.")
		return src
	}

	args := a.parseArgs(ctx)
	args = append(args, "--trace")
	args = append(args, "-")

	a.cfg.Logger.INFO.Println("Rendering", ctx.DocumentName, "with", path, "using asciidoctor args", args, "...")

	return internal.ExternallyRenderContent(a.cfg, ctx, src, path, args)
}

func (a *asciidocConverter) parseArgs(ctx converter.DocumentContext) []string {
	var cfg = a.cfg.MarkupConfig.AsciidocExt
	args := []string{}

	if asciidocext_config.BackendWhitelist[cfg.Backend] && cfg.Backend != asciidocext_config.Default.Backend {
		args = append(args, "-b", cfg.Backend)
	}

	for _, extension := range cfg.Extensions {
		if asciidocext_config.ExtensionsWhitelist[extension] != true {
			a.cfg.Logger.ERROR.Println("Unsupported asciidoctor extension was passed in.")
			continue
		}

		args = append(args, "-r", extension)
	}

	if cfg.WorkingFolderCurrent {
		contentDir := filepath.Dir(ctx.Filename)
		destinationDir := a.cfg.Cfg.GetString("destination")

		a.cfg.Logger.INFO.Println("destinationDir", destinationDir)
		if destinationDir == "" {
			a.cfg.Logger.ERROR.Println("markup.asciidocext.workingFolderCurrent requires hugo command option --destination to be set")
		}

		/* ToDo: RelPermalink patch for svg posts not working for asciidoctor-diagram
		  		postDir := ""
				page, ok := ctx.Document.(pageSubset)
				if ok {
					a.cfg.Logger.INFO.Println("path: ", page.RelPermalink())
					postDir = filepath.Base(page.RelPermalink())
				} else {
					a.cfg.Logger.ERROR.Println("unable to cast interface to pageSubset")
				}

				outDir, err := filepath.Abs(filepath.Join(destinationDir, filepath.Dir(ctx.DocumentName), postDir))
		*/
		outDir, err := filepath.Abs(filepath.Dir(filepath.Join(destinationDir, ctx.DocumentName)))

		if err != nil {
			a.cfg.Logger.ERROR.Println("asciidoctor outDir: ", err)
		}

		args = append(args, "--base-dir", contentDir, "-a", "outdir="+outDir)
	}

	if cfg.NoHeaderOrFooter {
		args = append(args, "--no-header-footer")
	} else {
		a.cfg.Logger.WARN.Println("asciidoctor parameter NoHeaderOrFooter is expected for correct html rendering")
	}

	if cfg.SectionNumbers != asciidocext_config.Default.SectionNumbers {
		args = append(args, "--section-numbers")
	}

	if cfg.Verbose != asciidocext_config.Default.Verbose {
		args = append(args, "-v")
	}

	if cfg.Trace != asciidocext_config.Default.Trace {
		args = append(args, "--trace")
	}

	if asciidocext_config.FailureLevelWhitelist[cfg.FailureLevel] && cfg.FailureLevel != asciidocext_config.Default.FailureLevel {
		args = append(args, "--failure-level", cfg.FailureLevel)
	}

	if asciidocext_config.SafeModeWhitelist[cfg.SafeMode] && cfg.SafeMode != asciidocext_config.Default.SafeMode {
		args = append(args, "--safe-mode", cfg.SafeMode)
	}

	return args
}

func getAsciidoctorExecPath() string {
	path, err := exec.LookPath("asciidoctor")
	if err != nil {
		return ""
	}
	return path
}

// Supports returns whether Asciidoctor is installed on this computer.
func Supports() bool {
	return getAsciidoctorExecPath() != ""
}

/*******************************************************************************
*
* Copyright 2018 Stefan Majewsky <majewsky@gmx.net>
*
* This program is free software: you can redistribute it and/or modify it under
* the terms of the GNU General Public License as published by the Free Software
* Foundation, either version 3 of the License, or (at your option) any later
* version.
*
* This program is distributed in the hope that it will be useful, but WITHOUT ANY
* WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
* A PARTICULAR PURPOSE. See the GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along with
* this program. If not, see <http://www.gnu.org/licenses/>.
*
*******************************************************************************/

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang-commonmark/markdown"
)

//SourceFile represents a Markdown source file.
type SourceFile struct {
	FilesystemPath string
	URLPath        string
}

//FindSourceFiles discovers all source files in the input directory.
func FindSourceFiles(inputDir string) ([]SourceFile, error) {
	var result []SourceFile

	baseDir := filepath.Join(inputDir, "spec")
	err := walk(baseDir, func(path string, fi os.FileInfo) {
		if fi.Mode().IsRegular() && strings.HasSuffix(path, ".md") {
			relativePath, _ := filepath.Rel(baseDir, path)
			result = append(result, newSourceFile(path, filepath.Join("std", relativePath)))
		}
	})
	if err != nil {
		return nil, err
	}

	baseDir = filepath.Join(inputDir, "website/pages")
	err = walk(baseDir, func(path string, fi os.FileInfo) {
		if fi.Mode().IsRegular() && strings.HasSuffix(path, ".md") {
			relativePath, _ := filepath.Rel(baseDir, path)
			result = append(result, newSourceFile(path, relativePath))
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

//Like filepath.Walk(), but don't pass errors to the callback.
func walk(root string, callback func(string, os.FileInfo)) error {
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		callback(path, fi)
		return nil
	})
}

//Called by FindSourceFiles().
func newSourceFile(path, relativePath string) SourceFile {
	//e.g.        path = "/path/to/vt6/spec/core/1.0.md"
	//and relativePath = "std/core/1.0.md" ("std" was added above already)

	//strip the ".md" suffix from the URL
	urlPath := strings.TrimSuffix(relativePath, ".md")

	//if the path ends in "index", that's just a trick for when we cannot spell
	//out the path otherwise, e.g. the page for URLPath = "/" is at
	//"website/pages/index.md" instead of "website/pages/.md" because that would
	//be a hidden file
	if urlPath == "index" {
		urlPath = "/"
	} else {
		urlPath = filepath.Join("/", strings.TrimSuffix(urlPath, "/index"))
	}

	return SourceFile{
		FilesystemPath: path,
		URLPath:        urlPath,
	}
}

//Render converts the Markdown from the source file to HTML and initializes a
//Page instance for this source file.
func (s SourceFile) Render() (Page, error) {
	contentBytes, err := ioutil.ReadFile(s.FilesystemPath)
	if err != nil {
		return Page{}, err
	}
	md := markdown.New(markdown.HTML(true))
	tokens := md.Parse(contentBytes)
	toc := CollectTableOfContents(tokens)
	contentHTML := md.RenderTokensToString(tokens)

	//recognize paragraphs starting with *Rationale:*
	contentHTML = strings.Replace(contentHTML,
		"\n<p><em>Rationale:</em>",
		"\n<p class=\"rationale\"><em>Rationale:</em>",
		-1,
	)
	//add link targets to headings
	contentHTML = InjectTargetsIntoHeadings(contentHTML, toc)

	//compile TikZ code into SVGs
	tikzOpening := `<pre><code class="language-tikz">`
	tikzClosing := `</code></pre>`
	tikzRx := regexp.MustCompile(tikzOpening + `(?s)(.*?)` + tikzClosing)
	err = nil
	contentHTML = tikzRx.ReplaceAllStringFunc(contentHTML, func(match string) string {
		match = strings.TrimPrefix(match, tikzOpening)
		match = strings.TrimSuffix(match, tikzClosing)
		match = html.UnescapeString(match)
		str, err2 := compileTikzPicture(match)
		if err == nil {
			err = err2
		}
		return str
	})
	if err != nil {
		return Page{}, err
	}

	//recognize draft marker ...
	fields := strings.SplitN(contentHTML, "\n", 2)
	firstLine := fields[0]
	isDraft := strings.TrimSpace(firstLine) == "<!-- draft -->"
	//... and remove it
	if isDraft {
		contentHTML = fields[1]
	}

	//recognize explicit title/description declaration...
	fields = strings.SplitN(contentHTML, "\n", 2)
	firstLine = fields[0]
	match := regexp.MustCompile(`^<!--\s*(\{.*\})\s*-->$`).FindStringSubmatch(firstLine)
	title, description := "", ""
	if match != nil {
		//...parse it...
		var data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		err := json.Unmarshal([]byte(match[1]), &data)
		if err != nil {
			return Page{}, fmt.Errorf(
				"read %s: unmarshal front matter failed: %s",
				s.FilesystemPath, err.Error(),
			)
		}
		title = data.Title
		description = data.Description

		//... and remove it
		contentHTML = fields[1]
	}

	//find page title, usually from leading heading
	if title == "" {
		if len(toc) > 0 && toc[0].IsPageTitle {
			title = toc[0].Caption
			// do we have a pair of title and description, e.g.
			// title = "vt6/posix1.0 - Platform integration on POSIX-compliant systems"
			fields := strings.SplitN(title, " - ", 2)
			if len(fields) == 2 {
				title = fields[0]
				description = fields[1]
			}
		} else {
			//last-resort fallback
			fmt.Fprintf(os.Stderr, "WARNING: cannot determine page title for %s\n", s.URLPath)
			title = strings.TrimPrefix(s.URLPath, "/")
		}
	}

	return Page{
		Path:                s.URLPath,
		Title:               title,
		Description:         description,
		IsDraft:             isDraft,
		ContentHTML:         template.HTML(contentHTML),
		TableOfContentsHTML: template.HTML(RenderTableOfContents(toc)),
	}, nil
}

//Takes in some LaTeX/TikZ source code and returns the rendered SVG.
func compileTikzPicture(code string) (svgCode string, returnErr error) {
	//split preamble from drawing code
	fields := regexp.MustCompile(`(?m)^---\s*$`).Split(code, 2)
	if len(fields) < 2 {
		return "", errors.New("cannot find preamble separator")
	}
	preamble := strings.TrimSpace(fields[0])
	drawingCode := strings.TrimSpace(fields[1])

	//create temp directory for compilation
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	tempDir := filepath.Join(
		os.TempDir(),
		"vt6-website-build-"+hex.EncodeToString(randomBytes),
	)
	err := os.MkdirAll(tempDir, 0700)
	if err != nil {
		return "", err
	}
	defer func() {
		if returnErr == nil {
			returnErr = os.RemoveAll(tempDir)
		}
	}()

	//prepare full LaTeX source file
	fullCode := "\\documentclass[tikz]{standalone}\n" +
		preamble + "\\begin{document}\\begin{tikzpicture}\n" +
		drawingCode + "\\end{tikzpicture}\\end{document}\n"
	err = ioutil.WriteFile(filepath.Join(tempDir, "picture.tex"), []byte(fullCode), 0600)
	if err != nil {
		return "", err
	}

	//run pdflatex
	cmd := exec.Command("pdflatex", "-interaction", "nonstopmode", "picture")
	cmd.Dir = tempDir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("exec pdflatex failed: %s", err.Error())
	}

	//run pdf2svg
	cmd = exec.Command("pdf2svg", "picture.pdf", "/dev/fd/1", "1")
	cmd.Dir = tempDir
	cmd.Stdin = nil
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("exec pdflatex failed: %s", err.Error())
	}
	return buf.String(), nil
}

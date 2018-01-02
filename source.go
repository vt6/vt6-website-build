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
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
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
		urlPath = "."
	} else {
		urlPath = strings.TrimSuffix(urlPath, "/index")
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
	contentHTML := markdown.New(markdown.HTML(true)).RenderToString(contentBytes)

	//recognize paragraphs starting with *Rationale:*
	contentHTML = strings.Replace(contentHTML,
		"\n<p><em>Rationale:</em>",
		"\n<p class=\"rationale\"><em>Rationale:</em>",
		-1,
	)

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

	if title == "" {
		title, description = extractTitle(s.URLPath, contentHTML)
	}
	return Page{
		Path:        s.URLPath,
		Title:       title,
		Description: description,
		IsDraft:     isDraft,
		ContentHTML: template.HTML(contentHTML),
	}, nil
}

func extractTitle(urlPath string, contentHTML string) (title, description string) {
	//get title/description from initial heading
	firstLine := strings.SplitN(contentHTML, "\n", 2)[0]
	if strings.HasPrefix(firstLine, "<h") {
		//remove all HTML tags, e.g.
		//before:  <h1><code>vt6/posix1.0</code> - Platform integration on POSIX-compliant systems</h1>
		// after:  vt6/posix1.0 - Platform integration on POSIX-compliant systems
		title = regexp.MustCompile(`</?\w+>`).ReplaceAllString(firstLine, "")
		// ^ This regex is ridiculously simple, but it catches all the tags
		// generated by the Markdown renderer. We don't need to cover all of HTML
		// here.

		// do we have a pair of title and description (such as in the example above?)
		fields := strings.SplitN(title, " - ", 2)
		if len(fields) == 2 {
			title = fields[0]
			description = fields[1]
		}
	}

	//last-resort fallback for title
	if title == "" {
		fmt.Fprintf(os.Stderr,
			"WARNING: cannot determine page title for %s\n",
			filepath.Join("/", urlPath),
		)
		title = urlPath
	}

	return
}

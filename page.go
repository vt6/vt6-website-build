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
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var pageTmpl *template.Template

func initPageTemplate(inputDir string) error {
	content, err := ioutil.ReadFile(filepath.Join(inputDir, "website/templates/page.html.tpl"))
	if err != nil {
		return err
	}
	pageTmpl, err = template.New("page").Parse(string(content))
	return err
}

//Page represents all the metadata and content of a HTML page on this website.
type Page struct {
	Path                string //e.g. "std/core/1.0"
	Title               string
	Description         string
	IsDraft             bool
	ContentHTML         template.HTML
	UpwardsNavigation   []NavigationLink
	DownwardsNavigation []NavigationLink
}

//WriteTo writes the HTML for this page to the corresponding path in the output
//directory.
func (p Page) WriteTo(outputDir string) error {
	p.Path = filepath.Clean(p.Path)

	var buf bytes.Buffer
	err := pageTmpl.Execute(&buf, p)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, strings.TrimPrefix(p.Path, "/"))
	err = os.MkdirAll(outputPath, 0777)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filepath.Join(outputPath, "index.html"),
		append(bytes.TrimSpace(buf.Bytes()), '\n'),
		0666,
	)
}

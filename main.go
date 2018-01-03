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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 3 {
		os.Stderr.Write([]byte("usage: vt6-website-build <path-to-vt6-repo> <path-to-output-dir>\n"))
		os.Exit(1)
	}

	//avoid duplication of error-printing code
	err := main2()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func main2() error {
	//first argument must be the VT6 repo, so we expect the "spec/" subdir with all the specs
	inputDir := os.Args[1]
	specDir := filepath.Join(inputDir, "spec")
	fi, err := os.Stat(specDir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New(specDir + ": not a directory")
	}

	//load templates
	err = initPageTemplate(inputDir)
	if err != nil {
		return err
	}

	//second argument must be a directory, but we create it on first run
	outputDir := os.Args[2]
	err = os.MkdirAll(outputDir, 0777)
	if err != nil {
		return err
	}

	//render source files
	sourceFiles, err := FindSourceFiles(inputDir)
	if err != nil {
		return err
	}
	pages := make([]*Page, len(sourceFiles))
	for idx, sourceFile := range sourceFiles {
		page, err := sourceFile.Render()
		if err != nil {
			return err
		}
		pages[idx] = &page
	}
	navTree := NewNavigationTree(sourceFiles)
	for _, page := range pages {
		page.AddNavigation(navTree)
	}

	//write resulting HTML pages to output directory
	for _, page := range pages {
		err = page.WriteTo(outputDir)
		if err != nil {
			return err
		}
	}

	//copy static assets
	return CopyAssets(
		filepath.Join(inputDir, "website/static"),
		filepath.Join(outputDir, "static"),
	)
}

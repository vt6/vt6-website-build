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
	"io/ioutil"
	"os"
	"path/filepath"
)

//CopyAssets copies all files from the input to the output directory.
func CopyAssets(inputDir, outputDir string) error {
	return filepath.Walk(inputDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.Mode().IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(inputDir, path)
		targetPath := filepath.Join(outputDir, relPath)

		err = os.MkdirAll(filepath.Dir(targetPath), 0777)
		if err != nil {
			return err
		}
		//copy files in such a way that symlinks are converted to regular files at the target
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(targetPath, buf, 0666)
	})
}

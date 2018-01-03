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
	"path/filepath"
	"strings"
)

//NavigationLink describes an entry in a Page's navigation bar.
type NavigationLink struct {
	URLPath string
	Caption string
}

//NavigationTree is a tree datastructure describing which pages exist and can
//be navigated to.
type NavigationTree struct {
	URLPath  string //path part of the URL represented by this instance (e.g. "/foo/bar")
	Parent   *NavigationTree
	Children map[string]*NavigationTree
	Exists   bool //whether there is a page at this point
}

//NewNavigationTree prepares a NavigationTree for the given set of source files.
func NewNavigationTree(sourceFiles []SourceFile) *NavigationTree {
	root := &NavigationTree{
		URLPath:  "/",
		Children: make(map[string]*NavigationTree),
	}

	for _, sf := range sourceFiles {
		ntLocate(root, sf.URLPath, true).Exists = true
	}

	return root
}

//AddNavigation populates the Page.UpwardsNavigation and Page.DownwardsNavigation.
func (p *Page) AddNavigation(root *NavigationTree) {
	//upwards navigation (this is computed based on the path only for now, might change later)
	//NOTE: We do not need a link to URLPath = "/" - The VT6 logo in the header serves that purpose.
	if p.Path == "/" {
		p.UpwardsNavigation = nil
	} else {
		pathElements := strings.Split(strings.TrimLeft(p.Path, "/"), "/")
		for idx := range pathElements {
			p.UpwardsNavigation = append(p.UpwardsNavigation, NavigationLink{
				URLPath: "/" + strings.Join(pathElements[0:idx+1], "/"),
				Caption: pathElements[idx],
			})
		}
	}

	//downwards navigation
	tree := ntLocate(root, p.Path, false)
	p.DownwardsNavigation = nil
	for _, child := range tree.Children {
		p.DownwardsNavigation = append(p.DownwardsNavigation,
			ntCollectDownwardsNav(tree, child)...,
		)
	}
}

func ntCollectDownwardsNav(tree *NavigationTree, child *NavigationTree) []NavigationLink {
	//when a child exists, we can navigate to it...
	if child.Exists {
		relPath, _ := filepath.Rel(tree.URLPath, child.URLPath)
		return []NavigationLink{{
			URLPath: child.URLPath,
			Caption: relPath,
		}}
	}

	//...otherwise we need to offer a way to navigate its children
	var links []NavigationLink
	for _, grandchild := range child.Children {
		links = append(links, ntCollectDownwardsNav(tree, grandchild)...)
	}
	return links
}

//ntLocate locates the subtree at the given path. If `createMissing` is true, it (and its
//parents) will be created on first use.
func ntLocate(root *NavigationTree, urlPath string, createMissing bool) *NavigationTree {
	if urlPath == "/" {
		return root
	}

	pathElements := strings.Split(strings.TrimLeft(urlPath, "/"), "/")

	current := root
	for _, elem := range pathElements {
		next, exists := current.Children[elem]
		if !exists {
			if !createMissing {
				return nil
			}
			next = &NavigationTree{
				URLPath:  filepath.Join(current.URLPath, elem),
				Parent:   current,
				Children: make(map[string]*NavigationTree),
			}
			current.Children[elem] = next
		}
		current = next
	}

	return current
}

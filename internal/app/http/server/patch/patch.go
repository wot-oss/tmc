// //go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

// PUT PATH VARIABLES TO BE REPLACED HERE !!!
// e.g. if path variables of routes need a regex pattern
var (
	routePathVarPatch = map[string]string{
		"{name}":       "{name:.+}",
		"{tmIDOrName}": "{tmIDOrName:.+}",
	}
)

const (
	fileName        = "server/server.gen.go"
	routeLinePrefix = "r.HandleFunc(options.BaseURL"
)

type lineData struct {
	idx      int
	oldValue string
	newValue string
}

func main() {
	var allLines, routes = readIn(fileName)
	patchRoutes(routes)
	patchLines(allLines, routes)
	writeFile(fileName, allLines)

	fmt.Printf("Patched %s successfully\n", fileName)
}

func readIn(fileName string) (lines []string, routePatches []*lineData) {

	f, err := os.Open(fileName)
	if err != nil {
		panic(fmt.Sprintf("cannot open file %s", fileName))
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var line string
	var cnt = 0
	for scanner.Scan() {
		line = scanner.Text()
		lines = append(lines, line)
		if strings.HasPrefix(strings.TrimSpace(line), routeLinePrefix) {
			routePatches = append(routePatches, &lineData{idx: cnt, oldValue: line})
		}
		cnt++
	}
	return lines, routePatches
}

func patchRoutes(routes []*lineData) {
	// collect the routes and sort them descending for correct route matching
	var s []string
	for _, p := range routes {
		s = append(s, p.oldValue)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(s)))

	// update line data and assign reordered routes
	for idx, p := range routes {
		p.newValue = s[idx]
		// replace path variables in route if needed
		for k, v := range routePathVarPatch {
			if strings.Contains(p.newValue, k) {
				p.newValue = strings.ReplaceAll(p.newValue, k, v)
			}
		}
	}
}

func patchLines(lines []string, patches []*lineData) {
	for _, p := range patches {
		lines[p.idx] = p.newValue
	}
}

func writeFile(fileName string, lines []string) {
	var b []byte
	for _, line := range lines {
		b = append(b, []byte(line+"\n")...)
	}
	err := utils.AtomicWriteFile(fileName, b, 0644)
	if err != nil {
		panic(fmt.Sprintf("could not write patched file %s, %v", fileName, err))
	}
}

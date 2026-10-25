package recipes

import (
	"fmt"
	"sort"
)

// Recipe represents a project scaffolding template with prerequisite validation
type Recipe struct {
	Flag               string
	Name               string
	Description        string
	CheckPrerequisites func() error
	Execute            func(projectPath, projectName string) error
}

var registry = make(map[string]Recipe)

// Register adds a recipe to the global registry
func Register(r Recipe) {
	registry[r.Flag] = r
}

// Get finds a recipe by its flag (e.g. "--node-ts")
func Get(flag string) (Recipe, bool) {
	r, exists := registry[flag]
	return r, exists
}

// List returns all registered recipes sorted by flag
func List() []Recipe {
	var list []Recipe
	for _, r := range registry {
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Flag < list[j].Flag
	})
	return list
}

// PrintAvailable prints all available recipes nicely formatted
func PrintAvailable() {
	for _, r := range List() {
		fmt.Printf("  %-18s %s\n", r.Flag, r.Description)
	}
}

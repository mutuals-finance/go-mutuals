package allocation

import (
	"context"
	"errors"
	"fmt"

	"github.com/mutuals/go-mutuals/service/persist"
)

// Claim is the single node structure.
type Claim struct {
	// Data Fields
	ID               persist.DBID
	Label            string
	Path             string
	ValidationData   persist.JSON
	DistributionData persist.JSON
	ValidationID     string
	DistributionID   string
	// Raw Inputs (used only for initial linking)
	ParentID *string
	ChildIDs []string
	// Pointer Links
	Parent   *Claim
	Children []*Claim
}

// Tree is a container for a flat claims list and root claims with pointers
type Tree struct {
	Claims []*Claim // Flat list of pointers to all claims
	Roots  []*Claim // Pointers to top-level claims
}

// NewTree parses the flat list, converts to pointers, and links references
func NewTree(inputs []Claim) (*Tree, error) {
	if len(inputs) == 0 {
		return &Tree{}, nil
	}

	t := &Tree{
		Claims: make([]*Claim, 0, len(inputs)),
		Roots:  make([]*Claim, 0),
	}

	labelMap := make(map[string]*Claim)
	allPtrs := make([]*Claim, len(inputs))

	// 1: create pointers and map by label
	for i := range inputs {
		ptr := &inputs[i]
		allPtrs[i] = ptr
		if ptr.Label != "" {
			labelMap[ptr.Label] = ptr
		}
	}

	// 2: Identify container-only nodes
	containerNodes := make(map[*Claim]bool)
	for _, ptr := range allPtrs {
		if ptr.ValidationID == "" && ptr.DistributionID == "" && len(ptr.ChildIDs) > 0 {
			containerNodes[ptr] = true
		}
	}

	// 3: determine what goes in Claims list (exclude containers)
	for _, ptr := range allPtrs {
		if !containerNodes[ptr] {
			t.Claims = append(t.Claims, ptr)
		}
	}

	// 4: Link parent-child relationships
	for _, ptr := range allPtrs {
		// Link Parent
		if ptr.ParentID != nil {
			if parent, ok := labelMap[*ptr.ParentID]; ok {
				ptr.Parent = parent
			}
		}

		// Link Children
		for _, childLabel := range ptr.ChildIDs {
			if child, ok := labelMap[childLabel]; ok {
				ptr.Children = append(ptr.Children, child)
				if child.Parent == nil {
					child.Parent = ptr
				}
			}
		}
	}

	// 5: Determine roots - nodes without parents OR nodes whose parent is a container
	for _, ptr := range allPtrs {
		if containerNodes[ptr] {
			continue // Skip container nodes themselves
		}

		if ptr.Parent == nil || containerNodes[ptr.Parent] {
			// This is a root: either no parent, or parent is a container
			t.Roots = append(t.Roots, ptr)
			// Clear the parent pointer if it was a container
			if ptr.Parent != nil && containerNodes[ptr.Parent] {
				ptr.Parent = nil
			}
		}
	}

	return t, nil
}

// Prepare generates ids and computes paths.
func (t *Tree) Prepare(ctx context.Context) error {
	for _, claim := range t.Claims {
		newID := persist.GenerateID()
		claim.ID = newID
		claim.Label = string(newID)
	}

	for _, root := range t.Roots {
		computePathRecursive(root, "")
	}

	return nil
}

func computePathRecursive(claim *Claim, parentPath string) {
	if parentPath == "" {
		claim.Path = claim.Label
	} else {
		claim.Path = parentPath + "." + claim.Label
	}

	for _, child := range claim.Children {
		computePathRecursive(child, claim.Path)
	}
}

// Validate performs basic sanity checks
func (t *Tree) Validate(ctx context.Context) error {
	for _, claim := range t.Claims {
		if hasCycle(claim, make(map[*Claim]bool)) {
			return errors.New(fmt.Sprintf("cycle detected starting at %s", claim.Label))
		}
	}
	return nil
}

func hasCycle(current *Claim, visited map[*Claim]bool) bool {
	if visited[current] {
		return true
	}
	visited[current] = true
	for _, child := range current.Children {
		if hasCycle(child, visited) {
			return true
		}
	}
	visited[current] = false
	return false
}

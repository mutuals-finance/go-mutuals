package allocation

import (
	"context"
	"errors"
	"fmt"

	"github.com/mutuals/go-mutuals/service/persist"
)

// ClaimValidator validates individual nodes
type ClaimValidator interface {
	Validate(ctx context.Context, treeCtx *Tree, node *Claim) error
	Name() string
}

// RequiredFieldsValidator ensures required fields are present
type RequiredFieldsValidator struct{}

func (v RequiredFieldsValidator) Name() string { return "required_fields" }

func (v RequiredFieldsValidator) Validate(ctx context.Context, tree *Tree, node *Claim) error {
	if len(node.Children) <= 0 && node.RecipientAddress.String() == "" {
		return errors.New(fmt.Sprintf("node %s: recipient address is required if no children", node.ID))
	}
	return nil
}

// RecipientAddressValidator validates recipient address format
type RecipientAddressValidator struct{}

func NewRecipientAddressValidator() *RecipientAddressValidator {
	return &RecipientAddressValidator{}
}

func (v *RecipientAddressValidator) Name() string { return "recipient_address" }

func (v *RecipientAddressValidator) Validate(ctx context.Context, treeCtx *Tree, node *Claim) error {
	// TODO : Implement address validation
	return nil
}

// ParentChildValidator validates parent-child relationships
type ParentChildValidator struct{}

func NewParentChildValidator() *ParentChildValidator {
	return &ParentChildValidator{}
}

func (v ParentChildValidator) Name() string { return "parent_child" }

func (v ParentChildValidator) Validate(ctx context.Context, tree *Tree, node *Claim) error {
	// Validate that parent reference exist
	if node.Parent != nil {
		if _, ok := tree.ClaimsByID[persist.DBID(*node.Parent)]; !ok {
			return errors.New(fmt.Sprintf("node %s: parent with id '%s' not found", node.ID, node.Parent))
		}
	}

	return nil
}

// TreeValidator validates the entire tree structure
type TreeValidator interface {
	Validate(ctx context.Context, treeCtx *Tree) error
	Name() string
}

// TreeCompletenessValidator ensures tree is complete and properly ordered
type TreeCompletenessValidator struct{}

func (v TreeCompletenessValidator) Name() string { return "tree_completeness" }

func (v TreeCompletenessValidator) Validate(ctx context.Context, tree *Tree) error {
	if tree.Count == 0 {
		return nil
	}

	// TODO check if properly ordered

	// Check for cycles (should not be possible with proper ID handling, but verify)
	if hasCycle(tree) {
		return errors.New("circular reference detected in tree structure")
	}

	return nil
}

func hasCycle(tree *Tree) bool {
	visited := make(map[persist.DBID]bool)
	recStack := make(map[persist.DBID]bool)

	var dfs func(nodeId persist.DBID) bool
	dfs = func(nodeId persist.DBID) bool {
		visited[nodeId] = true
		recStack[nodeId] = true

		for _, childId := range tree.ClaimsByID[nodeId].Children {
			if !visited[persist.DBID(childId)] {
				if dfs(persist.DBID(childId)) {
					return true
				}
			} else if recStack[persist.DBID(childId)] {
				return true
			}
		}

		recStack[nodeId] = false
		return false
	}

	for _, node := range tree.Flat {
		if !visited[node.ID] {
			if dfs(node.ID) {
				return true
			}
		}
	}

	return false
}

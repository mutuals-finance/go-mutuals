package allocation

import (
	"context"

	"github.com/mutuals/go-mutuals/service/persist"
)

// Claim represents a claim in the allocation tree
type Claim struct {
	ID               persist.DBID
	Label            string
	Path             string
	RecipientAddress *persist.Address
	Data             persist.JSON
	StateID          string
	StrategyID       string
	Parent           *string
	Children         []string
}

// Tree holds the full allocation tree
type Tree struct {
	RootClaims      []persist.DBID
	Flat            []Claim
	ClaimsByID      map[persist.DBID]*Claim
	ClaimsRefByID   map[persist.DBID][]persist.DBID
	UpdatedIDs      map[persist.DBID]persist.DBID
	Params          any
	Count           int
	claimValidators []ClaimValidator
	treeValidators  []TreeValidator
	claimProcessors []ClaimProcessor
	treeProcessors  []TreeProcessor
}

// NewTree constructs a tree from the flattened claim list
func NewTree(flat []Claim, opts ...TreeOption) (*Tree, error) {
	if len(flat) == 0 {
		return &Tree{}, nil
	}

	tree := &Tree{
		RootClaims:    []persist.DBID{},
		Flat:          flat,
		ClaimsByID:    make(map[persist.DBID]*Claim),
		ClaimsRefByID: make(map[persist.DBID][]persist.DBID),
		UpdatedIDs:    make(map[persist.DBID]persist.DBID),
		Params:        nil,
		Count:         len(flat),
		claimValidators: []ClaimValidator{
			RequiredFieldsValidator{},
			NewRecipientAddressValidator(),
			ParentChildValidator{},
		},
		treeValidators: []TreeValidator{
			TreeCompletenessValidator{},
		},
		claimProcessors: []ClaimProcessor{
			BaseClaimProcessor{},
			NewRecipientAddressProcessor(),
		},
		treeProcessors: []TreeProcessor{
			PathProcessor{},
		},
	}

	for _, opt := range opts {
		opt(tree)
	}

	for i := range tree.Flat {
		claim := &tree.Flat[i]
		tree.ClaimsByID[claim.ID] = claim
		if claim.Parent != nil {
			parentID := persist.DBID(*claim.Parent)
			tree.ClaimsRefByID[parentID] = append(tree.ClaimsRefByID[parentID], claim.ID)
		} else {
			tree.RootClaims = append(tree.RootClaims, claim.ID)
		}

		for _, childIDStr := range claim.Children {
			childID := persist.DBID(childIDStr)
			tree.ClaimsRefByID[childID] = append(tree.ClaimsRefByID[childID], claim.ID)
		}

	}

	return tree, nil
}

// Traverse performs depth-first traversal with a visitor function
func (tree *Tree) Traverse(fn func(claim *Claim, tree *Tree, args ...any) error, args ...any) error {
	for _, claim := range tree.Flat {
		if err := fn(&claim, tree, args...); err != nil {
			return err
		}
	}
	return nil
}

func (tree *Tree) Validate(ctx context.Context) (err error) {

	// Run tree validators
	for _, v := range tree.treeValidators {
		if err := v.Validate(ctx, tree); err != nil {
			return err
		}
	}

	// Run claim validators
	for _, claim := range tree.Flat {
		for _, v := range tree.claimValidators {
			if err := v.Validate(ctx, tree, &claim); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tree *Tree) Process(ctx context.Context) error {

	// Run claim processors
	for i := range tree.Flat {
		claim := &tree.Flat[i]
		for _, p := range tree.claimProcessors {
			if err := p.Process(ctx, tree, claim); err != nil {
				return err
			}
		}
	}

	// Run tree processors
	for _, p := range tree.treeProcessors {
		if err := p.Process(ctx, tree); err != nil {
			return err
		}
	}

	return nil
}

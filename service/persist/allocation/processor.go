package allocation

import (
	"context"

	"github.com/mutuals/go-mutuals/service/persist"
)

// ClaimProcessor processes individual claims
type ClaimProcessor interface {
	Process(ctx context.Context, tree *Tree, claim *Claim) error
	Name() string
}

// BaseClaimProcessor creates the basic DB params
type BaseClaimProcessor struct{}

func (p BaseClaimProcessor) Name() string { return "base_claim" }

func (p BaseClaimProcessor) Process(ctx context.Context, tree *Tree, claim *Claim) error {
	oldID := claim.ID
	oldIDStr := string(oldID)
	newID := persist.GenerateID()

	claim.ID = newID
	claim.Label = string(newID)

	tree.UpdatedIDs[oldID] = newID

	tree.ClaimsByID[newID] = claim

	if refs, exists := tree.ClaimsRefByID[oldID]; exists {
		for _, refID := range refs {
			var referencingClaim *Claim

			if refID == oldID {
				referencingClaim = claim
			} else {
				referencingClaim = tree.ClaimsByID[refID]

				if referencingClaim == nil {
					if newRefID, wasUpdated := tree.UpdatedIDs[refID]; wasUpdated {
						referencingClaim = tree.ClaimsByID[newRefID]
					}
				}
			}

			if referencingClaim == nil {
				continue
			}

			if referencingClaim.Parent != nil && *referencingClaim.Parent == oldIDStr {
				newIDStr := string(newID)
				referencingClaim.Parent = &newIDStr
			}

			for i, childID := range referencingClaim.Children {
				if childID == oldIDStr {
					referencingClaim.Children[i] = string(newID)
				}
			}
		}

		tree.ClaimsRefByID[newID] = refs
		delete(tree.ClaimsRefByID, oldID)
	}

	if oldID != newID {
		delete(tree.ClaimsByID, oldID)
	}

	for i, rootID := range tree.RootClaims {
		if rootID == oldID {
			tree.RootClaims[i] = newID
			break
		}
	}

	return nil
}

// RecipientAddressProcessor handles different address types
type RecipientAddressProcessor struct {
}

func NewRecipientAddressProcessor() *RecipientAddressProcessor {
	return &RecipientAddressProcessor{}
}

func (p *RecipientAddressProcessor) Name() string { return "recipient_address" }

func (p *RecipientAddressProcessor) Process(ctx context.Context, tree *Tree, claim *Claim) error {
	// TODO: Handle email/phone -> wallet resolution via Privy
	return nil
}

// ParamsBuildFunc is the callback function type for building params from a claim
type ParamsBuildFunc func(ctx context.Context, tree *Tree, claim *Claim) any
type ParamsBuildProcessor struct {
	buildFunc ParamsBuildFunc
}

func NewParamsBuildProcessor(fn ParamsBuildFunc) *ParamsBuildProcessor {
	return &ParamsBuildProcessor{buildFunc: fn}
}

func (p *ParamsBuildProcessor) Name() string { return "params_build" }

func (p *ParamsBuildProcessor) Process(ctx context.Context, tree *Tree, claim *Claim) error {
	if p.buildFunc != nil {
		tree.Params = p.buildFunc(ctx, tree, claim)
	}

	return nil
}

// TreeProcessor processes the entire tree
type TreeProcessor interface {
	Name() string
	Process(ctx context.Context, tree *Tree) error
}

// PathProcessor computes paths for all claims
type PathProcessor struct{}

func (p PathProcessor) Name() string { return "path_compute" }

func (p PathProcessor) Process(ctx context.Context, tree *Tree) error {
	for _, rootID := range tree.RootClaims {
		rootClaim := tree.ClaimsByID[rootID]
		computePath(tree, rootClaim, "")
	}
	return nil
}

func computePath(tree *Tree, claim *Claim, parentPath string) {
	if claim == nil {
		return
	}

	if parentPath == "" {
		claim.Path = claim.Label
	} else {
		claim.Path = parentPath + "." + claim.Label
	}

	for _, childId := range claim.Children {
		childClaim := tree.ClaimsByID[persist.DBID(childId)]
		if childClaim != nil {
			computePath(tree, childClaim, claim.Path)
		}
	}
}

// IPFSUploadProcessor uploads tree to IPFS (placeholder for future)
type IPFSUploadProcessor struct {
	// ipfsClient will be added later
}

func NewIPFSUploadProcessor() *IPFSUploadProcessor {
	return &IPFSUploadProcessor{}
}

func (p IPFSUploadProcessor) Name() string { return "ipfs_upload" }

func (p IPFSUploadProcessor) Process(ctx context.Context, tree *Tree) error {
	// TODO: Upload tree structure to IPFS
	// result.TreeHash = ipfsHash
	return nil
}

package allocation

// TreeOption configures the Tree
type TreeOption func(tree *Tree)

// WithClaimValidator adds a claim validator
func WithClaimValidator(v ClaimValidator) TreeOption {
	return func(tree *Tree) {
		tree.claimValidators = append(tree.claimValidators, v)
	}
}
 
// WithTreeValidator adds a tree validator
func WithTreeValidator(v TreeValidator) TreeOption {
	return func(tree *Tree) {
		tree.treeValidators = append(tree.treeValidators, v)
	}
}

// WithClaimProcessor adds a claim processor
func WithClaimProcessor(p ClaimProcessor) TreeOption {
	return func(tree *Tree) {
		tree.claimProcessors = append(tree.claimProcessors, p)
	}
}

// WithTreeProcessor adds a tree processor
func WithTreeProcessor(p TreeProcessor) TreeOption {
	return func(tree *Tree) {
		tree.treeProcessors = append(tree.treeProcessors, p)
	}
}

// WithParamsBuildFunc adds a params build processor with the given callback
func WithParamsBuildFunc(fn ParamsBuildFunc) TreeOption {
	return func(tree *Tree) {
		tree.claimProcessors = append(tree.claimProcessors, NewParamsBuildProcessor(fn))
	}
}

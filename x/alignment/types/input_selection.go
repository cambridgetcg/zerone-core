package types

import "context"

// InputSelection is the immutable, entry-point snapshot of the two adapter
// markers. It is not persisted and must not be cached between blocks.
type InputSelection struct {
	BankSupply        bool
	ActiveDomainCount bool
}

type inputSelectionKey struct{}

// WithInputSelection carries a successfully read selection to the sensors and
// their adapters without changing their pinned interfaces or implementations.
func WithInputSelection(ctx context.Context, selection InputSelection) context.Context {
	return context.WithValue(ctx, inputSelectionKey{}, selection)
}

// InputSelectionFromContext distinguishes an explicitly selected legacy path
// from a standalone caller that still needs to read its own marker.
func InputSelectionFromContext(ctx context.Context) (InputSelection, bool) {
	selection, ok := ctx.Value(inputSelectionKey{}).(InputSelection)
	return selection, ok
}

// BankSupplySelector and ActiveDomainCountSelector are implemented by the real
// adapters. Unwired dependencies and old sensor-only test doubles remain legacy.
type BankSupplySelector interface {
	AlignmentBankSupplyEnabled(context.Context) (bool, error)
}

type ActiveDomainCountSelector interface {
	ActiveDomainCountEnabled(context.Context) (bool, error)
}

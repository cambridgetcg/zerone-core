package branchflow

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput           = errors.New("invalid branch-flow input")
	ErrLimitExceeded          = errors.New("branch-flow limit exceeded")
	ErrInvalidGraph           = errors.New("invalid branch-flow graph")
	ErrReceiptAlreadyConsumed = errors.New("economic receipt already consumed")
	ErrInternalInvariant      = errors.New("branch-flow invariant failure")
)

const (
	CodeInvalidInput                = "INVALID_INPUT"
	CodeUnsupportedSchema           = "UNSUPPORTED_SCHEMA"
	CodeInvalidID                   = "INVALID_ID"
	CodeInvalidNodeMode             = "INVALID_NODE_MODE"
	CodeInvalidMilestone            = "INVALID_MILESTONE"
	CodeInvalidAmount               = "INVALID_AMOUNT"
	CodeLimitExceeded               = "LIMIT_EXCEEDED"
	CodeInvalidPolicySum            = "INVALID_POLICY_SUM"
	CodeInvalidPolicyDigest         = "INVALID_POLICY_DIGEST"
	CodeInvalidContinuation         = "INVALID_CONTINUATION"
	CodeInvalidDepth                = "INVALID_DEPTH"
	CodeDuplicateID                 = "DUPLICATE_ID"
	CodeUnknownNode                 = "UNKNOWN_NODE"
	CodeSelfEdge                    = "SELF_EDGE"
	CodeDuplicateEdge               = "DUPLICATE_EDGE"
	CodeZeroEdge                    = "ZERO_EDGE"
	CodeTooManyParents              = "TOO_MANY_PARENTS"
	CodeGraphCycle                  = "GRAPH_CYCLE"
	CodeTraversalLimit              = "TRAVERSAL_LIMIT"
	CodeInvalidCreditSum            = "INVALID_CREDIT_SUM"
	CodeInvalidImpactDisposition    = "INVALID_IMPACT_DISPOSITION"
	CodeInvalidReceiptKey           = "INVALID_RECEIPT_KEY"
	CodeDuplicateReceipt            = "DUPLICATE_RECEIPT"
	CodeReceiptAlreadyConsumed      = "RECEIPT_ALREADY_CONSUMED"
	CodeImpactNotLinked             = "IMPACT_NOT_LINKED"
	CodeAllocationLineLimit         = "ALLOCATION_LINE_LIMIT"
	CodeInternalConservationFailure = "INTERNAL_CONSERVATION_FAILURE"
)

// ValidationError provides a stable machine code and input path while still
// supporting errors.Is against a broad category.
type ValidationError struct {
	Code   string
	Path   string
	Detail string
	Cause  error
}

func (e *ValidationError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	}
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Detail)
}

func (e *ValidationError) Unwrap() error {
	if e.Cause == nil {
		return ErrInvalidInput
	}
	return e.Cause
}

func inputError(code, path, detail string) error {
	return &ValidationError{Code: code, Path: path, Detail: detail, Cause: ErrInvalidInput}
}

func limitError(code, path, detail string) error {
	return &ValidationError{Code: code, Path: path, Detail: detail, Cause: ErrLimitExceeded}
}

func graphError(code, path, detail string) error {
	return &ValidationError{Code: code, Path: path, Detail: detail, Cause: ErrInvalidGraph}
}

func receiptConsumedError(path, detail string) error {
	return &ValidationError{
		Code: CodeReceiptAlreadyConsumed, Path: path, Detail: detail,
		Cause: ErrReceiptAlreadyConsumed,
	}
}

func invariantError(detail string) error {
	return &ValidationError{
		Code: CodeInternalConservationFailure, Detail: detail,
		Cause: ErrInternalInvariant,
	}
}

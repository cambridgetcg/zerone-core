package types

import "errors"

// ErrHealthObservationFault names Q1's fail-closed registry traversal halt.
// Unlike ErrMarkerFault this is a sensor-input I/O fault, reached only on a
// scheduled observation with Q1 enabled. EndBlock returns it before writes.
var ErrHealthObservationFault = errors.New("alignment health observation registry fault")

package types

import (
	"encoding/binary"
)

const (
	ModuleName = "substrate_bridge"
	StoreKey   = ModuleName

	AuditBountyPoolModuleName = "useful_work_audit_bounty_pool"
)

var (
	LineageEdgePrefix                = []byte{0x80}
	LineageByUpstreamPrefix          = []byte{0x81}
	LineageByDownstreamPrefix        = []byte{0x82}
	LineageRoyaltyAccumulatorPrefix  = []byte{0x83}
	AdapterRegistrationPrefix        = []byte{0x84}
	ExternalAttestationPrefix        = []byte{0x85}
	AttestationByStatusPrefix        = []byte{0x86}
	PendingFactIndexPrefix           = []byte{0x87}
	AttestationPendingClaimsPrefix   = []byte{0x88}
	AdapterByStatusPrefix            = []byte{0x89}

	ParamsKey               = []byte{0x8A}
	AttestationIDCounterKey = []byte{0x8B}

	WitnessPendingRewardPrefix = []byte{0x8C}
	WitnessDeadlineIndexPrefix = []byte{0x8D}
)

func WitnessPendingRewardKey(attestationID string) []byte {
	return append(append([]byte{}, WitnessPendingRewardPrefix...), []byte(attestationID)...)
}

func WitnessDeadlineIndexKey(deadline uint64, attestationID string) []byte {
	key := append(append([]byte{}, WitnessDeadlineIndexPrefix...), BeUint64(deadline)...)
	return append(key, []byte(attestationID)...)
}

func AdapterKey(adapterID string) []byte {
	return append(append([]byte{}, AdapterRegistrationPrefix...), []byte(adapterID)...)
}

func AdapterByStatusKey(status uint8, adapterID string) []byte {
	key := append([]byte{}, AdapterByStatusPrefix...)
	key = append(key, status)
	key = append(key, []byte(adapterID)...)
	return key
}

func AttestationKey(attestationID string) []byte {
	return append(append([]byte{}, ExternalAttestationPrefix...), []byte(attestationID)...)
}

func AttestationByStatusKey(status uint8, attestationID string) []byte {
	key := append([]byte{}, AttestationByStatusPrefix...)
	key = append(key, status)
	key = append(key, []byte(attestationID)...)
	return key
}

func AttestationByStatusPrefixForStatus(status uint8) []byte {
	return append(append([]byte{}, AttestationByStatusPrefix...), status)
}

func PendingFactIndexKey(pendingClaimID string) []byte {
	return append(append([]byte{}, PendingFactIndexPrefix...), []byte(pendingClaimID)...)
}

func AttestationPendingClaimsKey(attestationID, claimID string) []byte {
	key := append([]byte{}, AttestationPendingClaimsPrefix...)
	key = append(key, []byte(attestationID)...)
	key = append(key, 0x00)
	key = append(key, []byte(claimID)...)
	return key
}

func AttestationPendingClaimsPrefixFor(attestationID string) []byte {
	key := append([]byte{}, AttestationPendingClaimsPrefix...)
	key = append(key, []byte(attestationID)...)
	key = append(key, 0x00)
	return key
}

func LineageEdgeKey(edgeID string) []byte {
	return append(append([]byte{}, LineageEdgePrefix...), []byte(edgeID)...)
}

func LineageByUpstreamKey(upstreamID, edgeID string) []byte {
	key := append([]byte{}, LineageByUpstreamPrefix...)
	key = append(key, []byte(upstreamID)...)
	key = append(key, 0x00)
	key = append(key, []byte(edgeID)...)
	return key
}

func LineageByUpstreamPrefixFor(upstreamID string) []byte {
	key := append([]byte{}, LineageByUpstreamPrefix...)
	key = append(key, []byte(upstreamID)...)
	key = append(key, 0x00)
	return key
}

func LineageByDownstreamKey(downstreamID, edgeID string) []byte {
	key := append([]byte{}, LineageByDownstreamPrefix...)
	key = append(key, []byte(downstreamID)...)
	key = append(key, 0x00)
	key = append(key, []byte(edgeID)...)
	return key
}

func LineageByDownstreamPrefixFor(downstreamID string) []byte {
	key := append([]byte{}, LineageByDownstreamPrefix...)
	key = append(key, []byte(downstreamID)...)
	key = append(key, 0x00)
	return key
}

func LineageRoyaltyAccumulatorKey(attestationID string) []byte {
	return append(append([]byte{}, LineageRoyaltyAccumulatorPrefix...), []byte(attestationID)...)
}

func EdgeID(upstreamID, downstreamID string) string {
	return upstreamID + "→" + downstreamID
}

func Be8(status uint8) []byte { return []byte{status} }

func BeUint64(v uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	return buf
}

package types

const (
	ModuleName = "evidence_mgmt"
	StoreKey   = "evid_mgmt"
)

// KV store key prefixes.
var (
	ParamsKey                  = []byte{0x00}
	EvidenceKeyPrefix          = []byte{0x01}
	VerificationKeyPrefix      = []byte{0x02}
	SubmitterIndexPrefix       = []byte{0x03}
	EvidenceCounterKey         = []byte{0x04}
	VerificationCounterKey     = []byte{0x05}
)

func EvidenceKey(id string) []byte {
	return append(EvidenceKeyPrefix, []byte(id)...)
}

func VerificationKey(id string) []byte {
	return append(VerificationKeyPrefix, []byte(id)...)
}

func VerificationByEvidencePrefix(evidenceID string) []byte {
	key := append(VerificationKeyPrefix, []byte(evidenceID)...)
	key = append(key, '/')
	return key
}

func VerificationByEvidenceKey(evidenceID, verificationID string) []byte {
	key := VerificationByEvidencePrefix(evidenceID)
	key = append(key, []byte(verificationID)...)
	return key
}

func SubmitterIndexKey(submitter, evidenceID string) []byte {
	key := append(SubmitterIndexPrefix, []byte(submitter)...)
	key = append(key, '/')
	key = append(key, []byte(evidenceID)...)
	return key
}

func SubmitterIndexKeyPrefix(submitter string) []byte {
	key := append(SubmitterIndexPrefix, []byte(submitter)...)
	key = append(key, '/')
	return key
}

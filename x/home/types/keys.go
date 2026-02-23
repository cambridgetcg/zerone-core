package types

const (
	// ModuleName is the home module's name.
	ModuleName = "home"

	// StoreKey is the store key for the home module.
	StoreKey = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey            = []byte{0x00}
	HomeKeyPrefix        = []byte{0x01}
	SessionKeyPrefix     = []byte{0x02}
	KeyRegKeyPrefix      = []byte{0x03}
	AlertKeyPrefix       = []byte{0x04}
	SpendingLimitPrefix  = []byte{0x05}
	HomeByOwnerPrefix    = []byte{0x06}
	HomeCounterKey       = []byte{0x07}
)

// HomeKey returns the store key for a home by ID.
func HomeKey(homeID string) []byte {
	return append(HomeKeyPrefix, []byte(homeID)...)
}

// SessionKey returns the store key for a session.
func SessionKey(homeID, sessionID string) []byte {
	return append(SessionKeyPrefix, []byte(homeID+"/"+sessionID)...)
}

// SessionPrefixKey returns the prefix key for all sessions of a home.
func SessionPrefixKey(homeID string) []byte {
	return append(SessionKeyPrefix, []byte(homeID+"/")...)
}

// KeyRegKey returns the store key for a key registration.
func KeyRegKey(homeID, keyHash string) []byte {
	return append(KeyRegKeyPrefix, []byte(homeID+"/"+keyHash)...)
}

// KeyRegPrefixKey returns the prefix key for all keys of a home.
func KeyRegPrefixKey(homeID string) []byte {
	return append(KeyRegKeyPrefix, []byte(homeID+"/")...)
}

// AlertKey returns the store key for an alert.
func AlertKey(homeID, alertID string) []byte {
	return append(AlertKeyPrefix, []byte(homeID+"/"+alertID)...)
}

// AlertPrefixKey returns the prefix key for all alerts of a home.
func AlertPrefixKey(homeID string) []byte {
	return append(AlertKeyPrefix, []byte(homeID+"/")...)
}

// SpendLimitKey returns the store key for a spending limit.
func SpendLimitKey(homeID, keyType string) []byte {
	return append(SpendingLimitPrefix, []byte(homeID+"/"+keyType)...)
}

// SpendLimitPrefixKey returns the prefix key for all spending limits of a home.
func SpendLimitPrefixKey(homeID string) []byte {
	return append(SpendingLimitPrefix, []byte(homeID+"/")...)
}

// HomeByOwnerKey returns the index key for homes by owner.
func HomeByOwnerKey(owner string) []byte {
	return append(HomeByOwnerPrefix, []byte(owner)...)
}

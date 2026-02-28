package keeper

import (
	"context"
	"encoding/json"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// DomainStats tracks the population of a domain for carrying capacity calculations.
type DomainStats struct {
	Domain      string `json:"domain"`
	ActiveCount uint64 `json:"active_count"`
	AtRiskCount uint64 `json:"at_risk_count"`
	TotalEnergy uint64 `json:"total_energy"`
	LastUpdated uint64 `json:"last_updated"` // block height
}

func (k Keeper) SetDomainStats(ctx context.Context, stats *DomainStats) {
	store := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(stats)
	_ = store.Set(types.DomainStatsKey(stats.Domain), bz)
}

func (k Keeper) GetDomainStats(ctx context.Context, domain string) (*DomainStats, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.DomainStatsKey(domain))
	if err != nil || bz == nil {
		return &DomainStats{Domain: domain}, false
	}
	var stats DomainStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return &DomainStats{Domain: domain}, false
	}
	return &stats, true
}

func (k Keeper) IncrementDomainFactCount(ctx context.Context, domain string, isActive bool, energy uint64) {
	stats, _ := k.GetDomainStats(ctx, domain)
	stats.Domain = domain
	if isActive {
		stats.ActiveCount++
	} else {
		stats.AtRiskCount++
	}
	stats.TotalEnergy += energy
	k.SetDomainStats(ctx, stats)
}

func (k Keeper) DecrementDomainFactCount(ctx context.Context, domain string, wasActive bool, energy uint64) {
	stats, _ := k.GetDomainStats(ctx, domain)
	stats.Domain = domain
	if wasActive {
		if stats.ActiveCount > 0 {
			stats.ActiveCount--
		}
	} else {
		if stats.AtRiskCount > 0 {
			stats.AtRiskCount--
		}
	}
	if energy > stats.TotalEnergy {
		stats.TotalEnergy = 0
	} else {
		stats.TotalEnergy -= energy
	}
	k.SetDomainStats(ctx, stats)
}

// TransitionDomainFactStatus updates stats when a fact moves between active/at-risk.
func (k Keeper) TransitionDomainFactStatus(ctx context.Context, domain string, toActive bool) {
	stats, _ := k.GetDomainStats(ctx, domain)
	stats.Domain = domain
	if toActive {
		if stats.AtRiskCount > 0 {
			stats.AtRiskCount--
		}
		stats.ActiveCount++
	} else {
		if stats.ActiveCount > 0 {
			stats.ActiveCount--
		}
		stats.AtRiskCount++
	}
	k.SetDomainStats(ctx, stats)
}

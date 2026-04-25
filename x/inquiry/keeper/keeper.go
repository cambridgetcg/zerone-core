package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/inquiry/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	authority    string

	bankKeeper      types.BankKeeper
	knowledgeKeeper types.KnowledgeKeeper // optional; nil = manual-resolve only
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bankKeeper types.BankKeeper,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		bankKeeper:   bankKeeper,
	}
}

func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) { k.knowledgeKeeper = kk }

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) Authority() string { return k.authority }

// ─── Params ──────────────────────────────────────────────────────────

func (k Keeper) GetParams(ctx context.Context) types.Params {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.ParamsKey)
	if err != nil || bz == nil {
		return *types.DefaultParams()
	}
	var p types.Params
	if err := k.cdc.Unmarshal(bz, &p); err != nil {
		return *types.DefaultParams()
	}
	return p
}

func (k Keeper) SetParams(ctx context.Context, p types.Params) error {
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, bz)
}

// ─── ID generation ───────────────────────────────────────────────────

func (k Keeper) NextInquiryID(ctx context.Context) (string, error) {
	st := k.storeService.OpenKVStore(ctx)
	bz, err := st.Get(types.NextInquirySeqKey)
	if err != nil {
		return "", err
	}
	cur := uint64(1)
	if bz != nil && len(bz) == 8 {
		cur = binary.BigEndian.Uint64(bz)
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, cur+1)
	if err := st.Set(types.NextInquirySeqKey, out); err != nil {
		return "", err
	}
	return fmt.Sprintf("inq-%d", cur), nil
}

func (k Keeper) NextAnswerID(ctx context.Context) (uint64, error) {
	st := k.storeService.OpenKVStore(ctx)
	bz, err := st.Get(types.NextAnswerIDKey)
	if err != nil {
		return 0, err
	}
	cur := uint64(1)
	if bz != nil && len(bz) == 8 {
		cur = binary.BigEndian.Uint64(bz)
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, cur+1)
	if err := st.Set(types.NextAnswerIDKey, out); err != nil {
		return 0, err
	}
	return cur, nil
}

// ─── Inquiries ───────────────────────────────────────────────────────

func inqKey(id string) []byte {
	return append(append([]byte{}, types.InquiryKeyPrefix...), []byte(id)...)
}

func byDomainKey(domain, id string) []byte {
	out := append([]byte{}, types.ByDomainPrefix...)
	out = append(out, []byte(domain)...)
	out = append(out, '/')
	return append(out, []byte(id)...)
}

func byAskerKey(asker, id string) []byte {
	out := append([]byte{}, types.ByAskerPrefix...)
	out = append(out, []byte(asker)...)
	out = append(out, '/')
	return append(out, []byte(id)...)
}

func byStatusKey(status types.InquiryStatus, id string) []byte {
	out := append([]byte{}, types.ByStatusPrefix...)
	out = append(out, byte(status))
	return append(out, []byte(id)...)
}

func (k Keeper) GetInquiry(ctx context.Context, id string) (*types.Inquiry, bool) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(inqKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var q types.Inquiry
	if err := k.cdc.Unmarshal(bz, &q); err != nil {
		return nil, false
	}
	return &q, true
}

// SetInquiry writes the inquiry and updates the byStatus index
// (removing any prior status index for the same id). Indexes by
// domain and asker are write-once on creation.
func (k Keeper) SetInquiry(ctx context.Context, q *types.Inquiry, prev *types.Inquiry) error {
	st := k.storeService.OpenKVStore(ctx)
	bz, err := k.cdc.Marshal(q)
	if err != nil {
		return err
	}
	if err := st.Set(inqKey(q.Id), bz); err != nil {
		return err
	}
	// Write-once indexes: only set if this is the first persistence.
	if prev == nil {
		if err := st.Set(byDomainKey(q.Domain, q.Id), []byte{1}); err != nil {
			return err
		}
		if err := st.Set(byAskerKey(q.Asker, q.Id), []byte{1}); err != nil {
			return err
		}
	}
	// Status index: remove old, add new (if status changed).
	if prev != nil && prev.Status != q.Status {
		_ = st.Delete(byStatusKey(prev.Status, q.Id))
	}
	if prev == nil || prev.Status != q.Status {
		if err := st.Set(byStatusKey(q.Status, q.Id), []byte{1}); err != nil {
			return err
		}
	}
	return nil
}

// IterateInquiriesByStatus walks the byStatus index for one status.
// Used by BeginBlocker to find OPEN/ANSWERED inquiries to resolve.
func (k Keeper) IterateInquiriesByStatus(ctx context.Context, status types.InquiryStatus, limit uint32, cb func(q *types.Inquiry) bool) error {
	st := k.storeService.OpenKVStore(ctx)
	prefix := append(append([]byte{}, types.ByStatusPrefix...), byte(status))
	it, err := st.Iterator(prefix, nil)
	if err != nil {
		return err
	}
	defer it.Close()
	count := uint32(0)
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(prefix) || !bytesEqual(key[:len(prefix)], prefix) {
			break
		}
		id := string(key[len(prefix):])
		q, ok := k.GetInquiry(ctx, id)
		if !ok {
			continue
		}
		if cb(q) {
			break
		}
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	return nil
}

func (k Keeper) IterateInquiriesByDomain(ctx context.Context, domain string, cb func(q *types.Inquiry) bool) error {
	return k.iterateIDPrefix(ctx, types.ByDomainPrefix, domain, func(id string) bool {
		q, ok := k.GetInquiry(ctx, id)
		if !ok {
			return false
		}
		return cb(q)
	})
}

func (k Keeper) IterateInquiriesByAsker(ctx context.Context, asker string, cb func(q *types.Inquiry) bool) error {
	return k.iterateIDPrefix(ctx, types.ByAskerPrefix, asker, func(id string) bool {
		q, ok := k.GetInquiry(ctx, id)
		if !ok {
			return false
		}
		return cb(q)
	})
}

func (k Keeper) iterateIDPrefix(ctx context.Context, prefix []byte, anchor string, cb func(id string) bool) error {
	st := k.storeService.OpenKVStore(ctx)
	full := append(append([]byte{}, prefix...), []byte(anchor)...)
	full = append(full, '/')
	it, err := st.Iterator(full, nil)
	if err != nil {
		return err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(full) || !bytesEqual(key[:len(full)], full) {
			break
		}
		id := string(key[len(full):])
		if cb(id) {
			break
		}
	}
	return nil
}

// IterateAllInquiries walks every inquiry. Used for genesis export.
func (k Keeper) IterateAllInquiries(ctx context.Context, cb func(q *types.Inquiry) bool) error {
	st := k.storeService.OpenKVStore(ctx)
	it, err := st.Iterator(types.InquiryKeyPrefix, nil)
	if err != nil {
		return err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(types.InquiryKeyPrefix) ||
			!bytesEqual(key[:len(types.InquiryKeyPrefix)], types.InquiryKeyPrefix) {
			break
		}
		var q types.Inquiry
		if err := k.cdc.Unmarshal(it.Value(), &q); err != nil {
			continue
		}
		if cb(&q) {
			break
		}
	}
	return nil
}

// ─── Answers ─────────────────────────────────────────────────────────

func answerKey(id uint64) []byte {
	out := append([]byte{}, types.AnswerKeyPrefix...)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return append(out, buf...)
}

func answersByInquiryKey(inqID string, ansID uint64) []byte {
	out := append([]byte{}, types.AnswersByInquiryPrefix...)
	out = append(out, []byte(inqID)...)
	out = append(out, '/')
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, ansID)
	return append(out, buf...)
}

func answersByClaimKey(claimID string) []byte {
	return append(append([]byte{}, types.AnswersByClaimPrefix...), []byte(claimID)...)
}

func (k Keeper) GetAnswer(ctx context.Context, id uint64) (*types.Answer, bool) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(answerKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var a types.Answer
	if err := k.cdc.Unmarshal(bz, &a); err != nil {
		return nil, false
	}
	return &a, true
}

func (k Keeper) SetAnswer(ctx context.Context, a *types.Answer) error {
	bz, err := k.cdc.Marshal(a)
	if err != nil {
		return err
	}
	st := k.storeService.OpenKVStore(ctx)
	if err := st.Set(answerKey(a.Id), bz); err != nil {
		return err
	}
	if err := st.Set(answersByInquiryKey(a.InquiryId, a.Id), []byte{1}); err != nil {
		return err
	}
	idBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(idBuf, a.Id)
	return st.Set(answersByClaimKey(a.ClaimId), idBuf)
}

func (k Keeper) ClaimAlreadyLinked(ctx context.Context, claimID string) bool {
	bz, err := k.storeService.OpenKVStore(ctx).Get(answersByClaimKey(claimID))
	return err == nil && bz != nil
}

func (k Keeper) IterateAnswersByInquiry(ctx context.Context, inqID string, cb func(a *types.Answer) bool) error {
	st := k.storeService.OpenKVStore(ctx)
	prefix := append(append([]byte{}, types.AnswersByInquiryPrefix...), []byte(inqID)...)
	prefix = append(prefix, '/')
	it, err := st.Iterator(prefix, nil)
	if err != nil {
		return err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(prefix) || !bytesEqual(key[:len(prefix)], prefix) {
			break
		}
		idBytes := key[len(prefix):]
		if len(idBytes) != 8 {
			continue
		}
		id := binary.BigEndian.Uint64(idBytes)
		a, ok := k.GetAnswer(ctx, id)
		if !ok {
			continue
		}
		if cb(a) {
			break
		}
	}
	return nil
}

// CountAnswers returns the number of answers attached to inquiryID.
// Used for the per-inquiry answer cap check.
func (k Keeper) CountAnswers(ctx context.Context, inqID string) uint32 {
	c := uint32(0)
	_ = k.IterateAnswersByInquiry(ctx, inqID, func(_ *types.Answer) bool {
		c++
		return false
	})
	return c
}

// ─── Genesis ─────────────────────────────────────────────────────────

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if gs == nil {
		return
	}
	if gs.Params != nil {
		_ = k.SetParams(ctx, *gs.Params)
	}
	for _, q := range gs.Inquiries {
		if q != nil {
			_ = k.SetInquiry(ctx, q, nil)
		}
	}
	for _, a := range gs.Answers {
		if a != nil {
			_ = k.SetAnswer(ctx, a)
		}
	}
	st := k.storeService.OpenKVStore(ctx)
	if gs.NextInquirySeq > 0 {
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, gs.NextInquirySeq)
		_ = st.Set(types.NextInquirySeqKey, out)
	}
	if gs.NextAnswerId > 0 {
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, gs.NextAnswerId)
		_ = st.Set(types.NextAnswerIDKey, out)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	gs := &types.GenesisState{Params: &params}
	st := k.storeService.OpenKVStore(ctx)
	if bz, err := st.Get(types.NextInquirySeqKey); err == nil && len(bz) == 8 {
		gs.NextInquirySeq = binary.BigEndian.Uint64(bz)
	} else {
		gs.NextInquirySeq = 1
	}
	if bz, err := st.Get(types.NextAnswerIDKey); err == nil && len(bz) == 8 {
		gs.NextAnswerId = binary.BigEndian.Uint64(bz)
	} else {
		gs.NextAnswerId = 1
	}
	_ = k.IterateAllInquiries(ctx, func(q *types.Inquiry) bool {
		gs.Inquiries = append(gs.Inquiries, q)
		return false
	})
	for _, q := range gs.Inquiries {
		_ = k.IterateAnswersByInquiry(ctx, q.Id, func(a *types.Answer) bool {
			gs.Answers = append(gs.Answers, a)
			return false
		})
	}
	return gs
}

// ─── helpers ─────────────────────────────────────────────────────────

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func currentBlock(ctx context.Context) uint64 {
	h := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if h < 0 {
		return 0
	}
	return uint64(h)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEvaluateForkWithFunded32ByteBankAddress(t *testing.T) {
	address := encodeWithChecksumConstant(t, "zrn", bytes.Repeat([]byte{0x71}, 32), bech32ChecksumConstant)
	for _, spelling := range []string{address, strings.ToUpper(address)} {
		t.Run(spelling, func(t *testing.T) {
			fixture := forkFixture(t)
			mutateGenesisModule(t, &fixture, "bank", func(bank map[string]any) {
				coins := []any{map[string]any{"denom": "uother", "amount": "7"}}
				bank["balances"] = []any{map[string]any{"address": spelling, "coins": coins}}
				bank["supply"] = coins
			})
			fixture.resealArtifactsAfterGenesisChange(t)
			before := append([]byte(nil), fixture.genesis.AppState...)
			if err := validateForkGenesisQuiescence(fixture.genesis.AppState, fixture.release.InitialHeight); err != nil {
				t.Fatalf("unrelated funded 32-byte bank owner rejected: %v", err)
			}
			inputs := fixture.forkInputs()
			report, err := evaluate(inputs)
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != decisionFork {
				t.Fatalf("unrelated funded 32-byte owner: decision=%s reasons=%v", report.Decision, report.ReasonCodes)
			}
			if err := verifyGateReportWithInputs(report, inputs); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, fixture.genesis.AppState) {
				t.Fatal("recovery verification rewrote committed bank state")
			}
		})
	}
}

func TestForkSchedulerBankAddressDomain(t *testing.T) {
	for _, length := range []int{1, 19, 20, 21, 31, 32, 33, 50, 51, 255} {
		for _, uppercase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d-bytes/uppercase=%t", length, uppercase), func(t *testing.T) {
				payload := bytes.Repeat([]byte{0x71}, length)
				address := encodeWithChecksumConstant(t, "zrn", payload, bech32ChecksumConstant)
				if uppercase {
					address = strings.ToUpper(address)
				}
				decoded, err := decodeForkBankAddress("bank owner", address)
				if err != nil || !bytes.Equal(decoded, payload) {
					t.Fatalf("bank owner did not decode to its complete payload: %v", err)
				}
				raw := fundedBankAddressJSON(t, address, "uother")
				before := append([]byte(nil), raw...)
				if err := validateForkSchedulerBankBalances(raw); err != nil {
					t.Fatalf("SDK-compatible bank owner rejected: %v", err)
				}
				if !bytes.Equal(raw, before) {
					t.Fatal("bank scan modified source spelling")
				}
			})
		}
	}
}

func TestForkSchedulerBankAddressRejectsMalformed(t *testing.T) {
	payload := bytes.Repeat([]byte{0x71}, 32)
	address := encodeWithChecksumConstant(t, "zrn", payload, bech32ChecksumConstant)
	paddedData, err := convertBits(payload, 8, 5, true)
	if err != nil {
		t.Fatal(err)
	}
	// The last group of a 32-byte payload has four zero padding bits.
	paddedData[len(paddedData)-1] |= 1
	badPadding := "zrn1"
	for _, value := range append(paddedData, createBech32Checksum("zrn", paddedData, bech32ChecksumConstant)...) {
		badPadding += string(bech32Charset[value])
	}
	cases := map[string]string{
		"empty string":        "",
		"empty payload":       encodeWithChecksumConstant(t, "zrn", nil, bech32ChecksumConstant),
		"256-byte payload":    encodeWithChecksumConstant(t, "zrn", bytes.Repeat([]byte{0x71}, 256), bech32ChecksumConstant),
		"oversized encoding":  strings.Repeat("q", 1024),
		"wrong HRP":           encodeWithChecksumConstant(t, "cosmos", payload, bech32ChecksumConstant),
		"validator HRP":       encodeWithChecksumConstant(t, "zrnvaloper", payload, bech32ChecksumConstant),
		"wrong checksum":      address[:len(address)-1] + replacementBech32Character(address[len(address)-1]),
		"Bech32m":             encodeWithChecksumConstant(t, "zrn", payload, bech32mChecksumConstant),
		"nonzero padding":     badPadding,
		"mixed case":          "Z" + address[1:],
		"leading whitespace":  " " + address,
		"trailing whitespace": address + " ",
		"non-ASCII":           "ſ" + address[1:],
		"invalid alphabet":    address[:len(address)-1] + "!",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateForkSchedulerBankBalances(fundedBankAddressJSON(t, value, "uother")); err == nil {
				t.Fatal("malformed bank owner accepted")
			}
		})
	}
}

func TestForkSchedulerBankBalancesCompareFullAddressBytes(t *testing.T) {
	for _, module := range []string{"schedule", "message_schedule"} {
		for _, uppercase := range []bool{false, true} {
			for _, denom := range []string{"uzrn", "uother"} {
				t.Run(fmt.Sprintf("%s/uppercase=%t/%s", module, uppercase, denom), func(t *testing.T) {
					address := schedulerModuleAccountAddress(module)
					if uppercase {
						address = strings.ToUpper(address)
					}
					err := validateForkSchedulerBankBalances(fundedBankAddressJSON(t, address, denom))
					if err == nil || !strings.Contains(err.Error(), "module account must have zero all-denom balance") {
						t.Fatalf("funded scheduler must be rejected by the balance guard, got %v", err)
					}
					// An unrelated 32-byte owner sharing the complete 20-byte
					// scheduler prefix is not the module account.
					longPayload := append(schedulerModuleAccountAddressBytes(module), bytes.Repeat([]byte{0x71}, 12)...)
					longAddress := encodeWithChecksumConstant(t, "zrn", longPayload, bech32ChecksumConstant)
					if uppercase {
						longAddress = strings.ToUpper(longAddress)
					}
					if err := validateForkSchedulerBankBalances(fundedBankAddressJSON(t, longAddress, denom)); err != nil {
						t.Fatalf("nonmatching 32-byte owner rejected: %v", err)
					}
				})
			}
			t.Run(fmt.Sprintf("%s/uppercase=%t/empty", module, uppercase), func(t *testing.T) {
				address := schedulerModuleAccountAddress(module)
				if uppercase {
					address = strings.ToUpper(address)
				}
				raw := mustJSON(t, map[string]any{"balances": []any{map[string]any{"address": address, "coins": []any{}}}})
				if err := validateForkSchedulerBankBalances(raw); err != nil {
					t.Fatalf("empty scheduler balance rejected: %v", err)
				}
			})
		}
	}
}

func TestBankAddressDomainDoesNotRelaxValidatorIdentity(t *testing.T) {
	for _, length := range []int{1, 19, 21, 32, 51, 255} {
		t.Run(fmt.Sprintf("%d-bytes", length), func(t *testing.T) {
			payload := bytes.Repeat([]byte{0x71}, length)
			consensus := encodeWithChecksumConstant(t, "zrnvalcons", payload, bech32ChecksumConstant)
			if _, err := decodeCanonicalBech32Address("consensus", consensus, "zrnvalcons"); err == nil {
				t.Fatal("non-20-byte consensus identity accepted")
			}
			operator := encodeWithChecksumConstant(t, "zrnvaloper", payload, bech32ChecksumConstant)
			if _, err := decodeZRNValoper(operator); err == nil {
				t.Fatal("non-20-byte operator identity accepted")
			}
		})
	}
	consensus := encodeWithChecksumConstant(t, "zrnvalcons", bytes.Repeat([]byte{0x71}, 20), bech32ChecksumConstant)
	if _, err := decodeCanonicalBech32Address("consensus", strings.ToUpper(consensus), "zrnvalcons"); err == nil {
		t.Fatal("uppercase consensus identity accepted")
	}
}

func fundedBankAddressJSON(t *testing.T, address, denom string) json.RawMessage {
	t.Helper()
	coins := []any{map[string]any{"denom": denom, "amount": "7"}}
	return mustJSON(t, map[string]any{
		"balances": []any{map[string]any{"address": address, "coins": coins}},
		"supply":   coins,
	})
}

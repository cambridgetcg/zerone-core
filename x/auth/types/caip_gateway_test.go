package types_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gwv2runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/zerone-chain/zerone/x/auth/types"
)

type accountIdentifierQueryServer struct {
	types.UnimplementedQueryServer
}

func (accountIdentifierQueryServer) AccountIdentifier(_ context.Context, req *types.QueryAccountIdentifierRequest) (*types.QueryAccountIdentifierResponse, error) {
	return &types.QueryAccountIdentifierResponse{
		Identifier: &types.ChainAccountIdentifier{
			Namespace:  "cosmos",
			Reference:  "zerone-2",
			RawChainId: "zerone-2",
			AccountId:  "cosmos:zerone-2:" + req.Address,
			Address:    req.Address,
		},
	}, nil
}

func TestAccountIdentifierGRPCGatewayRoute(t *testing.T) {
	mux := gwv2runtime.NewServeMux()
	if err := types.RegisterQueryHandlerServer(context.Background(), mux, accountIdentifierQueryServer{}); err != nil {
		t.Fatalf("failed to register query gateway: %v", err)
	}

	path := "/zerone/auth/v1/account_identifier/" + caipTestAddress
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Identifier struct {
			AccountID string `json:"accountId"`
			Address   string `json:"address"`
		} `json:"identifier"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode gateway response: %v", err)
	}
	if body.Identifier.AccountID != "cosmos:zerone-2:"+caipTestAddress {
		t.Fatalf("unexpected account ID %q", body.Identifier.AccountID)
	}
	if body.Identifier.Address != caipTestAddress {
		t.Fatalf("unexpected address %q", body.Identifier.Address)
	}

	postRequest := httptest.NewRequest(http.MethodPost, path, nil)
	postResponse := httptest.NewRecorder()
	mux.ServeHTTP(postResponse, postRequest)
	if postResponse.Code == http.StatusOK {
		t.Fatal("POST unexpectedly reached the read-only GET route")
	}
}

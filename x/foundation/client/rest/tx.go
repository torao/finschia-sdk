package rest

import (
	"net/http"

	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"

	"github.com/cosmos/cosmos-sdk/client"
)

func DummyRESTHandler(_ client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "foundation",
		Handler:  func(_ http.ResponseWriter, _ *http.Request) {},
	}
}

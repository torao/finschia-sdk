package client

import (
	"github.com/cosmos/cosmos-sdk/x/foundation/client/cli"
	"github.com/cosmos/cosmos-sdk/x/foundation/client/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var UpdateFoundationParamsProposalHandler = govclient.NewProposalHandler(cli.NewProposalCmdUpdateFoundationParams, rest.DummyRESTHandler)
var UpdateValidatorAuthsProposalHandler = govclient.NewProposalHandler(cli.NewProposalCmdUpdateValidatorAuths, rest.DummyRESTHandler)

package tendermint

import (
	"github.com/cosmos/cosmos-sdk/x/ibc/light-clients/99-tendermint/types"
)

// Name returns the IBC client name
func Name() string {
	return types.SubModuleName
}

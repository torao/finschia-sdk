package token

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTransfer{},
		&MsgTransferFrom{},
		&MsgApprove{},
		&MsgIssue{},
		&MsgGrant{},
		&MsgRevoke{},
		&MsgMint{},
		&MsgBurn{},
		&MsgBurnFrom{},
		&MsgModify{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

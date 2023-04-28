package keeper

import (
	"context"
	"github.com/line/lbm-sdk/x/rollup"
)

type daServer struct {
	keeper Keeper
}

func NewDAServer(keeper Keeper) rollup.DAServer {
	return &daServer{
		keeper: keeper,
	}
}

var _ rollup.DAServer = daServer{}

func (s daServer) Submit(c context.Context, req *rollup.SubmitRequest) (*rollup.SubmitResponse, error) {
	return &rollup.SubmitResponse{}, nil
}

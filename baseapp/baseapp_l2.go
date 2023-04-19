//go:build layer2
// +build layer2

package baseapp

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

func (app *BaseApp) GetAppHash(hash abci.RequestGetAppHash) abci.ResponseGetAppHash {
	//TODO implement me
	panic("implement me")
}

func (app *BaseApp) GenerateFraudProof(proof abci.RequestGenerateFraudProof) abci.ResponseGenerateFraudProof {
	//TODO implement me
	panic("implement me")
}

func (app *BaseApp) VerifyFraudProof(proof abci.RequestVerifyFraudProof) abci.ResponseVerifyFraudProof {
	//TODO implement me
	panic("implement me")
}

// +build vscc

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package plugindispatcher

import (
	"fmt"

	"github.com/Yunpeng-J/fabric-protos-go/common"
	"github.com/Yunpeng-J/fabric-protos-go/peer"
	commonerrors "github.com/Yunpeng-J/HLF-2.2/common/errors"
	"github.com/Yunpeng-J/HLF-2.2/common/flogging"
	validation "github.com/Yunpeng-J/HLF-2.2/core/handlers/validation/api"
	s "github.com/Yunpeng-J/HLF-2.2/core/handlers/validation/api/state"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/pkg/errors"
)

// ChannelResources provides access to channel artefacts or
// functions to interact with them
type ChannelResources interface {
	// GetMSPIDs returns the IDs for the application MSPs
	// that have been defined in the channel
	GetMSPIDs() []string
}

// LedgerResources provides access to ledger artefacts or
// functions to interact with them
type LedgerResources interface {
	// NewQueryExecutor gives handle to a query executor.
	// A client can obtain more than one 'QueryExecutor's for parallel execution.
	// Any synchronization should be performed at the implementation level if required
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

// LifecycleResources provides access to chaincode lifecycle artefacts or
// functions to interact with them
type LifecycleResources interface {
	// ValidationInfo returns the name and arguments of the validation plugin for the supplied
	// chaincode. The function returns two types of errors, unexpected errors and validation
	// errors. The reason for this is that this function is called from the validation code,
	// which needs to differentiate the two types of error to halt processing on the channel
	// if the unexpected error is not nil and mark the transaction as invalid if the validation
	// error is not nil.
	ValidationInfo(channelID, chaincodeName string, qe ledger.SimpleQueryExecutor) (plugin string, args []byte, unexpectedErr error, validationErr error)
}

// CollectionResources provides access to collection artefacts
type CollectionResources interface {
	// CollectionValidationInfo returns collection-level endorsement policy for the supplied chaincode.
	// The function returns two types of errors, unexpected errors and validation errors. The
	// reason for this is that this function is to be called from the validation code, which
	// needs to tell apart the two types of error to halt processing on the channel if the
	// unexpected error is not nil and mark the transaction as invalid if the validation error
	// is not nil.
	CollectionValidationInfo(chaincodeName, collectionName string, state s.State) (args []byte, unexpectedErr error, validationErr error)
}

// CollectionAndLifecycleResources provides access to resources
// about chaincodes and their lifecycle and collections and their
// policies
type CollectionAndLifecycleResources interface {
	LifecycleResources

	// CollectionValidationInfo is exactly like the method defined in CollectionResources but
	// also takes the channel ID.  This is necessary to determine if the org collection names are valid.
	CollectionValidationInfo(channelID, chaincodeName, collectionName string, state s.State) (args []byte, unexpectedErr error, validationErr error)
}

//go:generate mockery -dir . -name LifecycleResources -case underscore -output mocks/

var logger = flogging.MustGetLogger("committer.txvalidator")

// dispatcherImpl is the implementation used to call
// the validation plugin and validate block transactions
type dispatcherImpl struct {
	chainID         string
	cr              ChannelResources
	ler             LedgerResources
	lcr             LifecycleResources
	pluginValidator *PluginValidator
}

// New creates new plugin dispatcher
func New(chainID string, cr ChannelResources, ler LedgerResources, lcr LifecycleResources, pluginValidator *PluginValidator) *dispatcherImpl {
	return &dispatcherImpl{
		chainID:         chainID,
		cr:              cr,
		ler:             ler,
		lcr:             lcr,
		pluginValidator: pluginValidator,
	}
}

// Dispatch executes the validation plugin(s) for transaction
func (v *dispatcherImpl) Dispatch(seq int, payload *common.Payload, envBytes []byte, block *common.Block) (error, peer.TxValidationCode) {
	// ethereum: disable VSCC, return VALID directly
	return nil, peer.TxValidationCode_VALID
}

func (v *dispatcherImpl) invokeValidationPlugin(ctx *Context) error {
	logger.Debug("Validating", ctx, "with plugin")
	err := v.pluginValidator.ValidateWithPlugin(ctx)
	if err == nil {
		return nil
	}
	// If the error is a pluggable validation execution error, cast it to the common errors ExecutionFailureError.
	if e, isExecutionError := err.(*validation.ExecutionFailureError); isExecutionError {
		return &commonerrors.VSCCExecutionFailureError{Err: e}
	}
	// Else, treat it as an endorsement error.
	return &commonerrors.VSCCEndorsementPolicyError{Err: err}
}

func (v *dispatcherImpl) getCDataForCC(channelID, ccid string) (string, []byte, error) {
	qe, err := v.ler.NewQueryExecutor()
	if err != nil {
		return "", nil, errors.WithMessage(err, "could not retrieve QueryExecutor")
	}
	defer qe.Done()

	plugin, args, unexpectedErr, validationErr := v.lcr.ValidationInfo(channelID, ccid, qe)
	if unexpectedErr != nil {
		return "", nil, &commonerrors.VSCCInfoLookupFailureError{
			Reason: fmt.Sprintf("Could not retrieve state for chaincode %s, error %s", ccid, unexpectedErr),
		}
	}
	if validationErr != nil {
		return "", nil, validationErr
	}

	if plugin == "" {
		return "", nil, errors.Errorf("chaincode definition for [%s] is invalid, plugin field must be set", ccid)
	}

	if len(args) == 0 {
		return "", nil, errors.Errorf("chaincode definition for [%s] is invalid, policy field must be set", ccid)
	}

	return plugin, args, nil
}

// GetInfoForValidate gets the ChaincodeInstance(with latest version) of tx, validation plugin and policy
func (v *dispatcherImpl) GetInfoForValidate(chdr *common.ChannelHeader, ccID string) (string, []byte, error) {
	// obtain name of the validation plugin and the policy
	plugin, args, err := v.getCDataForCC(chdr.ChannelId, ccID)
	if err != nil {
		msg := fmt.Sprintf("Unable to get chaincode data from ledger for txid %s, due to %s", chdr.TxId, err)
		logger.Errorf(msg)
		return "", nil, err
	}
	return plugin, args, nil
}

// txWritesToNamespace returns true if the supplied NsRwSet
// performs a ledger write
func (v *dispatcherImpl) txWritesToNamespace(ns *rwsetutil.NsRwSet) bool {
	// check for public writes first
	if ns.KvRwSet != nil && len(ns.KvRwSet.Writes) > 0 {
		return true
	}

	// check for private writes for all collections
	for _, c := range ns.CollHashedRwSets {
		if c.HashedRwSet != nil && len(c.HashedRwSet.HashedWrites) > 0 {
			return true
		}

		// private metadata updates
		if c.HashedRwSet != nil && len(c.HashedRwSet.MetadataWrites) > 0 {
			return true
		}
	}

	if ns.KvRwSet != nil && len(ns.KvRwSet.MetadataWrites) > 0 {
		return true
	}

	return false
}

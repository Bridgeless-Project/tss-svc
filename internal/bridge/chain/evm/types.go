package evm

import (
	v1 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v1"
	v2 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v2"
	v3 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type EventType string

const (
	EventV1DepositedNative           EventType = "DepositedNativeV1"
	EventV1DepositedERC20            EventType = "DepositedERC20V1"
	EventV2DepositedNative           EventType = "DepositedNativeV2"
	EventV2DepositedERC20            EventType = "DepositedERC20V2"
	EventV1DepositedNativeAndSwapped EventType = "DepositedNativeAndSwappedV1"
	EventV1DepositedERC20AndSwapped  EventType = "DepositedERC20AndSwappedV1"
)

const (
	EventNameDepositedNative           = "DepositedNative"
	EventNameDepositedERC20            = "DepositedERC20"
	EventNameDepositedNativeAndSwapped = "BridgedNativeAndSwapped"
	EventNameDepositedERC20AndSwapped  = "DepositedERC20AndSwapped"
)

var EventNameToEventV1 = map[string]EventType{
	EventNameDepositedNative: EventV1DepositedNative,
	EventNameDepositedERC20:  EventV1DepositedERC20,
}

var EventNameToEventV2 = map[string]EventType{
	EventNameDepositedERC20:  EventV2DepositedERC20,
	EventNameDepositedNative: EventV2DepositedNative,
}

var EventNameToEventV3 = map[string]EventType{
	EventNameDepositedNativeAndSwapped: EventV1DepositedNativeAndSwapped,
	EventNameDepositedERC20AndSwapped:  EventV1DepositedERC20AndSwapped,
}

var EventToEventName = map[EventType]string{
	EventV1DepositedNative:           EventNameDepositedNative,
	EventV2DepositedNative:           EventNameDepositedNative,
	EventV1DepositedERC20:            EventNameDepositedERC20,
	EventV2DepositedERC20:            EventNameDepositedERC20,
	EventV1DepositedNativeAndSwapped: EventNameDepositedNativeAndSwapped,
	EventV1DepositedERC20AndSwapped:  EventNameDepositedERC20AndSwapped,
}

const (
	ContractVersionV1 string = "contract_v1"
	ContractVersionV2 string = "contract_v2"
	ContractVersionV3 string = "contract_v3"
)

type abiVersion struct {
	metadata        *bind.MetaData
	eventNameToType map[string]EventType
}

var supportedVersions = map[string]abiVersion{
	ContractVersionV1: {v1.BridgeMetaData, EventNameToEventV1},
	ContractVersionV2: {v2.BridgeMetaData, EventNameToEventV2},
	ContractVersionV3: {v3.BridgeMetaData, EventNameToEventV3},
}

package evm

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

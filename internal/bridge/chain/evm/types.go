package evm

type EventType string

const (
	EventV1DepositedNative EventType = "DepositedNativeV1"
	EventV1DepositedERC20  EventType = "DepositedERC20V1"
	EventV2DepositedNative EventType = "DepositedNativeV2"
	EventV2DepositedERC20  EventType = "DepositedERC20V2"
	EventV3DepositedNative EventType = "DepositedNativeV3"
	EventV3DepositedERC20  EventType = "DepositedERC20V3"
)

const (
	EventNameDepositedNative = "DepositedNative"
	EventNameDepositedERC20  = "DepositedERC20"
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
	EventNameDepositedNative: EventV3DepositedNative,
	EventNameDepositedERC20:  EventV3DepositedERC20,
}

var EventToEventName = map[EventType]string{
	EventV1DepositedNative: EventNameDepositedNative,
	EventV2DepositedNative: EventNameDepositedNative,
	EventV3DepositedNative: EventNameDepositedNative,
	EventV1DepositedERC20:  EventNameDepositedERC20,
	EventV2DepositedERC20:  EventNameDepositedERC20,
	EventV3DepositedERC20:  EventNameDepositedERC20,
}

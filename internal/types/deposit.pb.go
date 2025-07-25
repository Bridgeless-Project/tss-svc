// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: deposit.proto

package types

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type WithdrawalStatus int32

const (
	WithdrawalStatus_WITHDRAWAL_STATUS_UNSPECIFIED WithdrawalStatus = 0
	WithdrawalStatus_WITHDRAWAL_STATUS_PENDING     WithdrawalStatus = 1
	WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING  WithdrawalStatus = 2
	WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED   WithdrawalStatus = 3
	WithdrawalStatus_WITHDRAWAL_STATUS_FAILED      WithdrawalStatus = 4
	WithdrawalStatus_WITHDRAWAL_STATUS_INVALID     WithdrawalStatus = 5
)

// Enum value maps for WithdrawalStatus.
var (
	WithdrawalStatus_name = map[int32]string{
		0: "WITHDRAWAL_STATUS_UNSPECIFIED",
		1: "WITHDRAWAL_STATUS_PENDING",
		2: "WITHDRAWAL_STATUS_PROCESSING",
		3: "WITHDRAWAL_STATUS_PROCESSED",
		4: "WITHDRAWAL_STATUS_FAILED",
		5: "WITHDRAWAL_STATUS_INVALID",
	}
	WithdrawalStatus_value = map[string]int32{
		"WITHDRAWAL_STATUS_UNSPECIFIED": 0,
		"WITHDRAWAL_STATUS_PENDING":     1,
		"WITHDRAWAL_STATUS_PROCESSING":  2,
		"WITHDRAWAL_STATUS_PROCESSED":   3,
		"WITHDRAWAL_STATUS_FAILED":      4,
		"WITHDRAWAL_STATUS_INVALID":     5,
	}
)

func (x WithdrawalStatus) Enum() *WithdrawalStatus {
	p := new(WithdrawalStatus)
	*p = x
	return p
}

func (x WithdrawalStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (WithdrawalStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_deposit_proto_enumTypes[0].Descriptor()
}

func (WithdrawalStatus) Type() protoreflect.EnumType {
	return &file_deposit_proto_enumTypes[0]
}

func (x WithdrawalStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use WithdrawalStatus.Descriptor instead.
func (WithdrawalStatus) EnumDescriptor() ([]byte, []int) {
	return file_deposit_proto_rawDescGZIP(), []int{0}
}

type DepositIdentifier struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TxHash        string                 `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	TxNonce       int64                  `protobuf:"varint,2,opt,name=tx_nonce,json=txNonce,proto3" json:"tx_nonce,omitempty"`
	ChainId       string                 `protobuf:"bytes,3,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DepositIdentifier) Reset() {
	*x = DepositIdentifier{}
	mi := &file_deposit_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DepositIdentifier) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DepositIdentifier) ProtoMessage() {}

func (x *DepositIdentifier) ProtoReflect() protoreflect.Message {
	mi := &file_deposit_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DepositIdentifier.ProtoReflect.Descriptor instead.
func (*DepositIdentifier) Descriptor() ([]byte, []int) {
	return file_deposit_proto_rawDescGZIP(), []int{0}
}

func (x *DepositIdentifier) GetTxHash() string {
	if x != nil {
		return x.TxHash
	}
	return ""
}

func (x *DepositIdentifier) GetTxNonce() int64 {
	if x != nil {
		return x.TxNonce
	}
	return 0
}

func (x *DepositIdentifier) GetChainId() string {
	if x != nil {
		return x.ChainId
	}
	return ""
}

type WithdrawalIdentifier struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TxHash        *string                `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3,oneof" json:"tx_hash,omitempty"`
	ChainId       string                 `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WithdrawalIdentifier) Reset() {
	*x = WithdrawalIdentifier{}
	mi := &file_deposit_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *WithdrawalIdentifier) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WithdrawalIdentifier) ProtoMessage() {}

func (x *WithdrawalIdentifier) ProtoReflect() protoreflect.Message {
	mi := &file_deposit_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WithdrawalIdentifier.ProtoReflect.Descriptor instead.
func (*WithdrawalIdentifier) Descriptor() ([]byte, []int) {
	return file_deposit_proto_rawDescGZIP(), []int{1}
}

func (x *WithdrawalIdentifier) GetTxHash() string {
	if x != nil && x.TxHash != nil {
		return *x.TxHash
	}
	return ""
}

func (x *WithdrawalIdentifier) GetChainId() string {
	if x != nil {
		return x.ChainId
	}
	return ""
}

type TransferData struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	Sender           *string                `protobuf:"bytes,1,opt,name=sender,proto3,oneof" json:"sender,omitempty"`
	Receiver         string                 `protobuf:"bytes,2,opt,name=receiver,proto3" json:"receiver,omitempty"`
	DepositAmount    string                 `protobuf:"bytes,3,opt,name=deposit_amount,json=depositAmount,proto3" json:"deposit_amount,omitempty"`
	WithdrawalAmount string                 `protobuf:"bytes,4,opt,name=withdrawal_amount,json=withdrawalAmount,proto3" json:"withdrawal_amount,omitempty"`
	CommissionAmount string                 `protobuf:"bytes,5,opt,name=commission_amount,json=commissionAmount,proto3" json:"commission_amount,omitempty"`
	DepositAsset     string                 `protobuf:"bytes,6,opt,name=deposit_asset,json=depositAsset,proto3" json:"deposit_asset,omitempty"`
	WithdrawalAsset  string                 `protobuf:"bytes,7,opt,name=withdrawal_asset,json=withdrawalAsset,proto3" json:"withdrawal_asset,omitempty"`
	IsWrappedAsset   bool                   `protobuf:"varint,8,opt,name=is_wrapped_asset,json=isWrappedAsset,proto3" json:"is_wrapped_asset,omitempty"`
	DepositBlock     int64                  `protobuf:"varint,9,opt,name=deposit_block,json=depositBlock,proto3" json:"deposit_block,omitempty"`
	// used for EVM transfers
	Signature     *string `protobuf:"bytes,10,opt,name=signature,proto3,oneof" json:"signature,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TransferData) Reset() {
	*x = TransferData{}
	mi := &file_deposit_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TransferData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransferData) ProtoMessage() {}

func (x *TransferData) ProtoReflect() protoreflect.Message {
	mi := &file_deposit_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransferData.ProtoReflect.Descriptor instead.
func (*TransferData) Descriptor() ([]byte, []int) {
	return file_deposit_proto_rawDescGZIP(), []int{2}
}

func (x *TransferData) GetSender() string {
	if x != nil && x.Sender != nil {
		return *x.Sender
	}
	return ""
}

func (x *TransferData) GetReceiver() string {
	if x != nil {
		return x.Receiver
	}
	return ""
}

func (x *TransferData) GetDepositAmount() string {
	if x != nil {
		return x.DepositAmount
	}
	return ""
}

func (x *TransferData) GetWithdrawalAmount() string {
	if x != nil {
		return x.WithdrawalAmount
	}
	return ""
}

func (x *TransferData) GetCommissionAmount() string {
	if x != nil {
		return x.CommissionAmount
	}
	return ""
}

func (x *TransferData) GetDepositAsset() string {
	if x != nil {
		return x.DepositAsset
	}
	return ""
}

func (x *TransferData) GetWithdrawalAsset() string {
	if x != nil {
		return x.WithdrawalAsset
	}
	return ""
}

func (x *TransferData) GetIsWrappedAsset() bool {
	if x != nil {
		return x.IsWrappedAsset
	}
	return false
}

func (x *TransferData) GetDepositBlock() int64 {
	if x != nil {
		return x.DepositBlock
	}
	return 0
}

func (x *TransferData) GetSignature() string {
	if x != nil && x.Signature != nil {
		return *x.Signature
	}
	return ""
}

var File_deposit_proto protoreflect.FileDescriptor

const file_deposit_proto_rawDesc = "" +
	"\n" +
	"\rdeposit.proto\x12\adeposit\"b\n" +
	"\x11DepositIdentifier\x12\x17\n" +
	"\atx_hash\x18\x01 \x01(\tR\x06txHash\x12\x19\n" +
	"\btx_nonce\x18\x02 \x01(\x03R\atxNonce\x12\x19\n" +
	"\bchain_id\x18\x03 \x01(\tR\achainId\"[\n" +
	"\x14WithdrawalIdentifier\x12\x1c\n" +
	"\atx_hash\x18\x01 \x01(\tH\x00R\x06txHash\x88\x01\x01\x12\x19\n" +
	"\bchain_id\x18\x02 \x01(\tR\achainIdB\n" +
	"\n" +
	"\b_tx_hash\"\xa3\x03\n" +
	"\fTransferData\x12\x1b\n" +
	"\x06sender\x18\x01 \x01(\tH\x00R\x06sender\x88\x01\x01\x12\x1a\n" +
	"\breceiver\x18\x02 \x01(\tR\breceiver\x12%\n" +
	"\x0edeposit_amount\x18\x03 \x01(\tR\rdepositAmount\x12+\n" +
	"\x11withdrawal_amount\x18\x04 \x01(\tR\x10withdrawalAmount\x12+\n" +
	"\x11commission_amount\x18\x05 \x01(\tR\x10commissionAmount\x12#\n" +
	"\rdeposit_asset\x18\x06 \x01(\tR\fdepositAsset\x12)\n" +
	"\x10withdrawal_asset\x18\a \x01(\tR\x0fwithdrawalAsset\x12(\n" +
	"\x10is_wrapped_asset\x18\b \x01(\bR\x0eisWrappedAsset\x12#\n" +
	"\rdeposit_block\x18\t \x01(\x03R\fdepositBlock\x12!\n" +
	"\tsignature\x18\n" +
	" \x01(\tH\x01R\tsignature\x88\x01\x01B\t\n" +
	"\a_senderB\f\n" +
	"\n" +
	"_signature*\xd4\x01\n" +
	"\x10WithdrawalStatus\x12!\n" +
	"\x1dWITHDRAWAL_STATUS_UNSPECIFIED\x10\x00\x12\x1d\n" +
	"\x19WITHDRAWAL_STATUS_PENDING\x10\x01\x12 \n" +
	"\x1cWITHDRAWAL_STATUS_PROCESSING\x10\x02\x12\x1f\n" +
	"\x1bWITHDRAWAL_STATUS_PROCESSED\x10\x03\x12\x1c\n" +
	"\x18WITHDRAWAL_STATUS_FAILED\x10\x04\x12\x1d\n" +
	"\x19WITHDRAWAL_STATUS_INVALID\x10\x05B6Z4github.com/Bridgeless-Project/tss-svc/internal/typesb\x06proto3"

var (
	file_deposit_proto_rawDescOnce sync.Once
	file_deposit_proto_rawDescData []byte
)

func file_deposit_proto_rawDescGZIP() []byte {
	file_deposit_proto_rawDescOnce.Do(func() {
		file_deposit_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_deposit_proto_rawDesc), len(file_deposit_proto_rawDesc)))
	})
	return file_deposit_proto_rawDescData
}

var file_deposit_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_deposit_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_deposit_proto_goTypes = []any{
	(WithdrawalStatus)(0),        // 0: deposit.WithdrawalStatus
	(*DepositIdentifier)(nil),    // 1: deposit.DepositIdentifier
	(*WithdrawalIdentifier)(nil), // 2: deposit.WithdrawalIdentifier
	(*TransferData)(nil),         // 3: deposit.TransferData
}
var file_deposit_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_deposit_proto_init() }
func file_deposit_proto_init() {
	if File_deposit_proto != nil {
		return
	}
	file_deposit_proto_msgTypes[1].OneofWrappers = []any{}
	file_deposit_proto_msgTypes[2].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_deposit_proto_rawDesc), len(file_deposit_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_deposit_proto_goTypes,
		DependencyIndexes: file_deposit_proto_depIdxs,
		EnumInfos:         file_deposit_proto_enumTypes,
		MessageInfos:      file_deposit_proto_msgTypes,
	}.Build()
	File_deposit_proto = out.File
	file_deposit_proto_goTypes = nil
	file_deposit_proto_depIdxs = nil
}

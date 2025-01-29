package zano

type SignedTransaction struct {
	UnsignedTransaction
	Signature string
}

type UnsignedTransaction struct {
	ExpectedTxHash string
	FinalizedTx    string
	Data           string
}

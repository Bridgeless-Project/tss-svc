package connector

const (
	txCodeSuccess       = 0
	txCodeWrongSequence = 32
)

func ApproximateGasLimit(gasUsed uint64) uint64 {
	return uint64(float64(gasUsed) * 1.5)
}

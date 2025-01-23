package session

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	KeygenSessionPrefix = "KEYGEN"
	SignSessionPrefix   = "SIGN"
)

type SigningSessionParams struct {
	Id        string
	StartTime time.Time
	Threshold int
}

func GetKeygenSessionIdentifier(sessionId string) string {
	return fmt.Sprintf("%s_%s", KeygenSessionPrefix, sessionId)
}

func GetDefaultSigningSessionIdentifier(sessionId string) string {
	return fmt.Sprintf("%s_%s", SignSessionPrefix, sessionId)
}

func GetConcreteSigningSessionIdentifier(chainId string, sessionId int64) string {
	return fmt.Sprintf("%s_%s_%d", SignSessionPrefix, chainId, sessionId)
}

func IncrementSessionIdentifier(id string) string {
	vals := strings.Split(id, "_")
	if len(vals) != 3 {
		return id
	}

	val, err := strconv.ParseInt(vals[2], 10, 64)
	if err != nil {
		return id
	}

	return fmt.Sprintf("%s_%s_%d", vals[0], vals[1], val+1)
}

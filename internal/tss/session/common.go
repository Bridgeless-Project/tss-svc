package session

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"

	"github.com/hyle-team/tss-svc/internal/tss"
)

const (
	KeygenSessionPrefix = "KEYGEN"
	SignSessionPrefix   = "SIGN"
)

type SigningSessionParams struct {
	tss.SessionParams
	ChainId string
}

func GetKeygenSessionIdentifier(sessionId int64) string {
	return fmt.Sprintf("%s_%d", KeygenSessionPrefix, sessionId)
}

func GetDefaultSigningSessionIdentifier(sessionId int64) string {
	return fmt.Sprintf("%s_%d", SignSessionPrefix, sessionId)
}

func GetSessionId(sessionIdentifier string) (int64, error) {
	str := strings.Split(sessionIdentifier, "_")[2]
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get session identifier")
	}
	return id, nil
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

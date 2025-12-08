package session

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
)

const (
	KeygenSessionPrefix  = "KEYGEN"
	SignSessionPrefix    = "SIGN"
	ReshareSessionPrefix = "RESHARE"
)

type Params struct {
	Id        int64     `fig:"session_id,required"`
	StartTime time.Time `fig:"start_time,required"`
	Threshold int       `fig:"threshold,required"`
}

type SigningParams struct {
	Params
	ChainId string
}

func ParamsFromSigningSessionInfo(info *p2p.SigningSessionInfo) SigningParams {
	return SigningParams{
		Params: Params{
			Id:        info.Id + 1, // expect the next session id to be the current id + 1
			StartTime: time.UnixMilli(info.NextSessionStartTime),
			Threshold: int(info.Threshold),
		},
		ChainId: info.ChainId,
	}
}

func ToSigningSessionInfo(
	signingSessionId string,
	nextSessionStartTime *time.Time,
	threshold int,
	chainId string,
) *p2p.SigningSessionInfo {
	sessInfo := &p2p.SigningSessionInfo{
		Id:        GetSessionId(signingSessionId),
		Threshold: int64(threshold),
		ChainId:   chainId,
	}

	if nextSessionStartTime != nil {
		sessInfo.NextSessionStartTime = nextSessionStartTime.UnixMilli()
	}

	return sessInfo
}

func GetSessionId(sessionIdentifier string) int64 {
	str := strings.Split(sessionIdentifier, "_")[2]
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func GetKeygenSessionIdentifier(sessionId int64) string {
	return fmt.Sprintf("%s_%d", KeygenSessionPrefix, sessionId)
}

func GetReshareSessionIdentifier(sessionId int64) string {
	return fmt.Sprintf("%s_%d", ReshareSessionPrefix, sessionId)
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

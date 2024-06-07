package serverstream

import (
	"strconv"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"github.com/gin-gonic/gin"
)

func ParseStringStreamRequest(c *gin.Context) (*tgsbpb.ServerStreamStringRequest, error) {
	incomingRequest := &tgsbpb.ServerStreamStringRequest{}
	incomingRequest.Value = c.Query("value")

	sendPeriodSeconds, _ := strconv.ParseUint(c.Query("sendPeriodSeconds"), 10, 32)
	incomingRequest.SendPeriodSeconds = uint32(sendPeriodSeconds)

	sendErrorAtNthResponse, _ := strconv.ParseUint(c.Query("sendErrorAtNthResponse"), 10, 32)
	incomingRequest.SendErrorAtNthResponse = uint32(sendErrorAtNthResponse)

	closeAtNthResponse, _ := strconv.ParseUint(c.Query("closeAtNthResponse"), 10, 32)
	incomingRequest.CloseAtNthResponse = uint32(closeAtNthResponse)

	return incomingRequest, nil
}

package status

import "github.com/haveachin/infrared/internal/pkg/java/protocol"

const ServerBoundRequestPacketID byte = 0x00

type ServerBoundRequest struct{}

func (pk ServerBoundRequest) Marshal() protocol.Packet {
	return protocol.MarshalPacket(
		ServerBoundRequestPacketID,
	)
}

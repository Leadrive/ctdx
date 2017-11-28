package packet

import (
	"net"
	"bytes"
	"encoding/binary"

	"github.com/datochan/gcom/cnet"
	gbytes "github.com/datochan/gcom/bytes"
	"github.com/datochan/gcom/utils"
	"github.com/datochan/gcom/crypto"
	"github.com/datochan/gcom/logger"
)

// 只做最简单实现: IPacketProtocol 接口
type TdxPacketProtocolImpl struct {
	packetBuffer  *gbytes.LittleEndianStreamImpl   // 封包接收的缓冲区
}

func NewDefaultProtocol() *TdxPacketProtocolImpl {
	return &TdxPacketProtocolImpl{gbytes.NewLittleEndianStream(make([]byte, 1024*1024*5, 1024*1024*5))}
}

func (tdx *TdxPacketProtocolImpl) recvToBuffer(s *cnet.Session) (int, error){
	tmpBuffer := make([]byte, 1024*5, 1024*5)

	recvLen, err := s.RawConn().Read(tmpBuffer[:])

	if recvLen <= 0 || err != nil {
		return 0, err
	}
	err = tdx.packetBuffer.WriteBuff(tmpBuffer[:recvLen])

	return recvLen, err
}

/**
 * 直接返回所有封包内容，不做任何处理
 */
func (tdx *TdxPacketProtocolImpl) ReadPacket(s *cnet.Session) (interface{}, error) {
	var newBuffer bytes.Buffer
	var header ResponseHeader

	headerSize := utils.SizeStruct(ResponseHeader{})

	if tdx.packetBuffer.Length() < headerSize {
		// 如果buffer的数据小于一个包头则需要开始从tdx获取
		_, err := tdx.recvToBuffer(s)
		if tdx.packetBuffer.Length() < headerSize || nil != err {
			return nil, err
		}
	}

	// 预读并解析包头信息
	tmpHeader, _ := tdx.packetBuffer.PeekBuff(headerSize)

	newBuffer.Write(tmpHeader)
	err := binary.Read(&newBuffer, binary.LittleEndian, &header)

	if nil != err { return nil, err }
	bodySize := int(header.BodyLength)

	if tdx.packetBuffer.Length() < bodySize+headerSize {
		// 如果buffer中的数据不完整则继续从tdx中获取
		_, err := tdx.recvToBuffer(s)
		if tdx.packetBuffer.Length() < bodySize+headerSize || nil != err {
			return nil, err
		}
	}

	// buffer中有完整的封包
	tdx.packetBuffer.ReadBuff(headerSize)
	pkgBody, _ := tdx.packetBuffer.ReadBuff(int(header.BodyLength))

	if header.IsCompress & 0x10 != 0{
		pkgBody = crypto.ZLibUnCompress(pkgBody)
	}

	// 清理掉已经处理的数据
	tdx.packetBuffer.CleanBuff()

	return ResponseNode{header, pkgBody}, nil
}

/**
 * 组包方法
 */
func (tdx *TdxPacketProtocolImpl) BuildPacket(pkgNode interface{}) []byte {
	requestNode := pkgNode.(RequestNode)
	byteHeader := GenerateHeader(requestNode.EventId, requestNode.CmdId,
		uint16(utils.SizeStruct(requestNode.RawData)), requestNode.IsRaw, requestNode.Index)

	return gbytes.BytesCombine(byteHeader, requestNode.RawData.([]byte))
}

func (tdx *TdxPacketProtocolImpl) SendPacket(conn net.Conn, buff []byte) error {
	_, err := conn.Write(buff)

	if err != nil {
		logger.Error("发生错误了, 错误信息: ", err)
	}

	return err
}
package packet

import (
	"time"
	"bytes"
	"strconv"
	"math/rand"
	"encoding/hex"
	"encoding/binary"

	"github.com/datochan/gcom/utils"
	"github.com/datochan/gcom/crypto"
	"github.com/datochan/gcom/logger"
)

// 通达信通讯封包包头结构
type header struct {
	flag  byte      // 固定为0x0C
	index uint16    // idx(先随机，发现特殊值再单独处理)
	cmdId uint16    // 具体命令的子标识
	isRaw byte      // 是否是未压缩的原始封包(已知封包全是0,待发现特殊值再特殊封装)
	bodyLength uint16        // H: 分别是封包长度(封包长度+2字节)
	bodyMaxLength  uint16    // H: 解压所需要的空间大小(已知两个相等待发现特殊值再特殊处理)
	eventId        uint16    // H: 事件标识, 靠此字段可以确定封包的类别
}

/**
 * 生成封包包头
 */
func GenerateHeader(eventId uint16, cmdId uint16, pkgLen uint16, isRaw byte, idx uint16) []byte {
	if idx == 0 { idx = uint16(rand.Intn(0x7fff)) }

	newBuffer := new(bytes.Buffer)
	rand.Seed(time.Now().UnixNano())

	binary.Write(newBuffer, binary.LittleEndian, header{0x0C,idx,cmdId,isRaw,pkgLen+2,pkgLen+2,eventId})
	return newBuffer.Bytes()
}

/**
 * 请求封包的基本包装
 */
type RequestNode struct{
	EventId   uint16
	CmdId     uint16
	IsRaw     byte
	Index     uint16
	RawData   interface{}    // 原始请求封包体的原始数据
}

type deviceInfo struct {
	unknown1    [110]byte // 0
	unknown2    uint32    // 0x01040000
	unknown3    uint32    // 0
	mainVersion float32
	coreVersion float32
	unknown4    uint32     // 0
	unknown5    [47]byte   // 0
	macAddr     [12]byte   // rand
	unknown6    [89]byte   // 0
}

func GenerateDeviceNode(mainVersion , coreVersion float32) RequestNode {
	var newBuffer bytes.Buffer
	var reqNode RequestNode
	reqNode.EventId = 0x0B
	reqNode.CmdId = 0x007B

	macAddr := [12]byte{}
	copy(macAddr[:], []byte(utils.RandomMacAddress()))
	deviceInfo := deviceInfo{[110]byte{}, 0x01040000, 0, mainVersion, coreVersion, 0,[47]byte{}, macAddr, [89]byte{}}
	binary.Write(&newBuffer, binary.LittleEndian, deviceInfo)

	pkgBuffer := crypto.Blowfish(newBuffer.Bytes())

	reqNode.RawData = pkgBuffer

	return reqNode
}

func GenerateMarketInitInfo() RequestNode {
	var reqNode RequestNode
	reqNode.EventId = 0x000D
	reqNode.CmdId = 0x0094
	reqNode.IsRaw = 1
	reqNode.RawData = []byte{01}

	return reqNode
}

type marketStockCount struct {
	market      uint16 // 深圳0, 上海1
	currentDate uint32 // 当前日期 yyyymmdd
}

func GenerateMarketStockCount(market int) RequestNode {
	var newBuffer bytes.Buffer
	var reqNode RequestNode
	reqNode.EventId = 0x044E
	reqNode.IsRaw = 1

	if market == 0 {
		// 深圳行情服务器
		reqNode.CmdId = 0x006B
	} else {
		//
		reqNode.CmdId = 0x006C
	}
	currentDate,_ :=strconv.Atoi(time.Now().Format("20060102"))

	binary.Write(&newBuffer, binary.LittleEndian, marketStockCount{uint16(market), uint32(currentDate)})
	reqNode.RawData = newBuffer.Bytes()

	return reqNode
}

// 请求服务器公告信息
func GenerateNotice() RequestNode {
	var reqNode RequestNode
	reqNode.EventId = 0x0FDB
	reqNode.CmdId = 0x0099
	reqNode.IsRaw = 1
	rawData, _ := hex.DecodeString("7464786C6576656C320000AE47E940040000000000000000000000000003")
	reqNode.RawData = rawData

	return reqNode
}

// 请求股票基础信息
type marketStockBase struct {
	market		uint16 // 深圳0, 上海1
	stockOffset		uint16 // 要获取的股票信息偏移
}

func GenerateMarketStockBase(market uint16, offset uint16) RequestNode {
	var newBuffer bytes.Buffer
	var reqNode RequestNode
	reqNode.EventId = 0x0450
	reqNode.IsRaw = 1

	if market == 0 {
		// 深圳行情服务器
		reqNode.CmdId = 0x006D
	} else {
		//
		reqNode.CmdId = 0x006E
	}

	binary.Write(&newBuffer, binary.LittleEndian, marketStockBase{market, offset})
	reqNode.RawData = newBuffer.Bytes()

	return reqNode
}

type StockBonus struct {
	Market byte
	Code   [6]byte
}

func GenerateStockBonus(stocks []StockBonus, index uint16) RequestNode {
	var newBuffer bytes.Buffer
	var reqNode RequestNode
	reqNode.CmdId = 0x0076
	reqNode.EventId = 0x000F
	reqNode.IsRaw = 1
	reqNode.Index = index

	count := uint16(len(stocks))
	err := binary.Write(&newBuffer, binary.LittleEndian, count)
	if nil != err {
		logger.Error("generate stock bonus err %v", err)
	}

	binary.Write(&newBuffer, binary.LittleEndian, stocks)
	reqNode.RawData = newBuffer.Bytes()

	return reqNode
}

// 日线行情信息结构
type stockHistoryItem struct {
	market   uint16     // 0: 深圳; 1: 上海
	code     [6]byte
	start    uint32
	end      uint32
	unknown1 uint16
}

func GenerateStockDayItem(market uint16, code string, start, end uint32, index uint16) RequestNode {
	var newBuffer bytes.Buffer
	var reqNode RequestNode
	var byCode [6]byte
	reqNode.CmdId = 0x0087
	reqNode.EventId = 0x0FCD
	reqNode.IsRaw = 1
	reqNode.Index = index

	copy(byCode[:], []byte(code))

	binary.Write(&newBuffer, binary.LittleEndian, stockHistoryItem{market, byCode, start, end, 0x0004})
	reqNode.RawData = newBuffer.Bytes()

	return reqNode
}

func GenerateStockMinsItem(market uint16, code string, start, end uint32, index uint16) RequestNode {
    var newBuffer bytes.Buffer
	var reqNode RequestNode
	var byCode [6]byte
    reqNode.CmdId = 0x008D
	reqNode.EventId = 0x0FCD
	reqNode.IsRaw = 1
	reqNode.Index = index

	copy(byCode[:], []byte(code))

	binary.Write(&newBuffer, binary.LittleEndian, stockHistoryItem{market, byCode, start, end, 0})
	reqNode.RawData = newBuffer.Bytes()

	return reqNode
}

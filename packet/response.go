package packet

/**
 * 应答封包的包头结构
 */
type ResponseHeader struct {
    PacketFlag uint32        // 封包标识
    IsCompress  byte         // 封包是否被压缩
    Index uint16             // 索引(股票索引)
    CmdId uint16             //
    Unknown1  byte           // 未知标识
    EventId    uint16        // 事件标识, 靠此字段可以确定封包的类别
    BodyLength uint16        // 分别是封包长度
    BodyMaxLength  uint16    // 解压所需要的空间大小
}

/**
 * 响应封包的基本包装
 */
type ResponseNode struct{
    ResponseHeader           // 收到的封包头信息
    RawData   interface{}    // 原始相应封包体的原始数据
}

/**
 * 市场最后交易的数据信息
 */
type MarketInitInfo struct {
    Unknown1   [10]uint32 // 10L: unknown
    Unknown2   uint16     // H: unknown
    DateSZ     uint32     // L: 深最后交易日期
    LastSZFlag uint32     // L: 深最后交易Flag
    DateSH     uint32     // L: 沪最后交易日期
    LastSHFlag uint32     // L: 沪最后交易Flag
    Unknown3   uint32     // L: unknown
    Unknown4   uint16     // H: unknown
    Unknown5   uint32     // L: unknown
    ServerName [21]byte   // 21s: 服务器名称
    DomainUrl  [18]byte   // 18s: domain
}

/**
 * 股票基础信息
 */
type StockBaseItem struct {
    Code         [6]byte    // 股票代码
    Unknown1     uint16     // 未知 固定0x64
    Name         [8]byte    // 股票名称
    Unknown2     uint32     // 未知
    Unknown3     byte       // 未知 固定0x02
    Price        float32    // 价格(昨收)
    Bonus1       uint16     // 用于计算权息数据
    Bonus2       uint16     // 权息数量
}

/**
 * 股票权息数据结构
 */
type StockBonusItem struct {
    Market       byte    // B: 市场(market): 0深, 1沪
    Code         [6]byte // 6s: 股票代码(code)
    Unknown1     byte    // B: 股票代码的0结束符(python解析麻烦,所以单独解析出来不使用)
    Date         int32   // L: 日期(date)
    Type         byte    // B: 分红配股类型(type): 1标识除权除息, 2: 配送股上市; 3: 非流通股上市; 4:未知股本变动; 5: 股本变动,6: 增发新股, 7: 股本回购, 8: 增发新股上市, 9:转配股上市
    Money        float32 // 送现金
    Price        float32 // 配股价
    Count        float32 // 送股数
    Rate         float32 // 配股比例
}

/**
 * 日线数据结构
 */
//B1 A2 33 01 BD 03 00 00 C3 03 00 00 BB 03 00 00 C0 03 00 00 71 BB 13 4E 75 B9 D9 03 00 00 01 00
//B2 A2 33 01 C0 03 00 00 C0 03 00 00 B0 03 00 00 BB 03 00 00 7C 5C 3C 4E 4A 00 F2 04 00 00 01 00
//----------- ----------- ----------- ----------- ----------- ----------- ----------- -----------
//日期(十进制) 开*100       高*100      低*100      收*100      成交额(float) 成交量
type StockDayItem struct {
     Date		uint32
     Open       uint32
     High       uint32
     Low        uint32
     Close      uint32
     Amount     uint32
     Volume     uint32
     Unknown1   uint32
}

/**
 * 五分钟线结构
 */
//97 63 3F 02 D7 A3 35 42 3D 0A 37 42 D7 A3 35 42 33 33 36 42 B0 9E B4 49 90 7E 00 00 00 00 00 00
//----- ----- ----------- ----------- ----------- ----------- ----------- ----------- -----------
//date time  open         high       low         close       amount      volume      unkown
type StockMinsItem struct {
    Date	   uint16
    Time       uint16
    Open       float32
    High       float32
    Low        float32
    Close      float32
    Amount     float32
    Volume     uint32
    Unknown1   uint32
}

/**
 * 财报数据
 */
type ReportHeader struct {  // 3h1H3L
    Unknown1 [3]uint16
    MaxCount uint16         // 财报最大记录数
    Unknown2 [3]uint32
}

type ReportItem struct {    //6s1c1L
    Code       [6]byte
    Unknown1   byte
    Foa        uint32
}

type ReportData struct {
    Prices    [264]float32
}

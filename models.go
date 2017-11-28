package ctdx


type LastTradeModel struct {
	ServerName string
	Domain     string
	SZDate     uint32
	SZFlag     uint32
	SZCount    uint32
	SHDate     uint32
	SHFlag     uint32
	SHCount    uint32
}

// 股票列表数据的文件结构
type StockBaseModel struct {
	Code         string  // 股票代码
	Name         string  // 股票名称
	Market       int     // 所属市场，0深交所，1上交所
	Unknown1     int     // 未知 固定0x64
	Unknown2     int     // 未知
	Unknown3     int     // 未知 固定0x02
	Price        float64 // 价格(昨收)
	Bonus1       int     // 用于计算权息数据
	Bonus2       int     // 权息数量
}

// 股票权息数据的文件结构
type StockBonusModel struct {
	Code         string  // 股票代码
	Date         int     // 日期
	Market       int     // 所属市场，0深交所，1上交所
	Type         int     // 分红配股类型(type): 1标识除权除息, 2: 配送股上市; 3: 非流通股上市; 4:未知股本变动; 5: 股本变动,6: 增发新股, 7: 股本回购, 8: 增发新股上市, 9:转配股上市
	Money        float64 // 送现金
	Price        float64 // 配股价
	Count        float64 // 送股数
	Rate         float64 // 配股比例
}

// 日线数据的文件结构
type StockDayModel struct {
    Date	   int
	Open       float64
	Low        float64
	High       float64
    Close      float64
	Volume     int
    Amount     float64
}

// 五分钟线数据的文件结构
type StockMinsModel struct {
    Date	   int
    Time	   string
	Open       float64
	Low        float64
	High       float64
    Close      float64
	Volume     int
    Amount     float64
}

// 历史ST股列表
type StockSTModel struct {
	Date       int
	Code       string
	Name       string
	Flag       string
}



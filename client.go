package ctdx

import (
	"os"
	"fmt"
	"strconv"

	"github.com/kniren/gota/series"
	"github.com/kniren/gota/dataframe"

	"github.com/datochan/gcom/utils"
	"github.com/datochan/gcom/logger"
	"github.com/datochan/gcom/cnet"

	"github.com/datochan/ctdx/comm"
	pkg "github.com/datochan/ctdx/packet"
)

const (
	stockBonusFinishedIdx = 0x1100   // 权息数据获取结束的标识符
)

type TdxClient struct {
	session     *cnet.SyncSession
	dispatcher  *CTdxDispatcher

	bonusFinishedChan   chan int   // 用于更新权息数据时同步已处理的数据

	Finished    chan interface{}
	Configure   comm.IConfigure
	MainVersion float32		// 软件版本 = 7.29
	CoreVersion float32		// 数据引擎版本 = 5.895
	lastTrade   LastTradeModel

	stockBaseDF dataframe.DataFrame
	stockbonusDF   dataframe.DataFrame
}

func NewDefaultTdxClient(configure comm.IConfigure) *TdxClient {
	return &TdxClient{MainVersion:7.29, CoreVersion:5.895, Configure:configure, Finished:make(chan interface{})}
}

/**
 * 关闭连接
 */
func (client *TdxClient) Close() { client.session.Close() }

/**
 * 与服务器建立TCP连接
 */
func (client *TdxClient) Conn(){
	var err error
	swProtocol := pkg.NewDefaultProtocol()
	client.dispatcher = NewCTdxDispatcher()

	client.session, err = cnet.NewSyncSession("tcp", client.Configure.GetTdx().Server.DataHost,
		swProtocol, client.dispatcher.HandleProc, 0)
	if err != nil {
		logger.Error("创建服务器链接失败,err: %v", err)
		os.Exit(0)
		return
	}

	client.session.SetCloseCallback(func(*cnet.Session) {
		logger.Info("服务器链接已关闭!")
		os.Exit(0)
	})

	client.session.Start()

	// 注册设备信息
	client.session.Send(pkg.GenerateDeviceNode(client.MainVersion, client.CoreVersion))

	// 设置市场最后交易信息
	lastHQInfo := pkg.GenerateMarketInitInfo()
	client.dispatcher.AddHandler(uint32(lastHQInfo.EventId), client.OnMarketInitInfo)
	client.session.Send(lastHQInfo)

	// 深交所中股债基数量
	hqServer := pkg.GenerateMarketStockCount(0)
	client.dispatcher.AddHandler(uint32(hqServer.EventId), client.OnStockCount)
	client.session.Send(hqServer)

	// 上交所中股债基数量
	hqServer = pkg.GenerateMarketStockCount(1)
	client.dispatcher.AddHandler(uint32(hqServer.EventId), client.OnStockCount)
	client.session.Send(hqServer)

	// 请求券商公告信息
	client.session.Send(pkg.GenerateNotice())
}

/**
 * 更新股票基础信息
 */
func (client *TdxClient) UpdateStockBase(){
	logger.Info("开始更新深交所股债基列表信息...")
	stockBase := pkg.GenerateMarketStockBase(0, 0)
	client.dispatcher.AddHandler(uint32(stockBase.EventId), client.OnStockBase)

	for idx := 0;uint32(idx) < client.lastTrade.SZCount; idx += 0x03E8 {
		stockBase = pkg.GenerateMarketStockBase(0, uint16(idx))
		client.session.Send(stockBase)
	}

	logger.Info("开始更新上交所股债基列表信息...")
	// 更新上交所股债基列表信息
	for idx := 0;uint32(idx) < client.lastTrade.SHCount; idx += 0x03E8 {
		stockBase = pkg.GenerateMarketStockBase(1, uint16(idx))
		client.session.Send(stockBase)
	}
}

func  (client *TdxClient)updateBonus(df *dataframe.DataFrame){
	// 筛选掉已经处理过的数据
	var row map[string]interface{}
	var stockBonus []pkg.StockBonus
	var finishedIdx int
	var idx int

	logger.Info("开始接收高送转数据...")
	filterDf := utils.ReIndex(df)

CONTINUE:
	filterDf = filterDf.Filter(dataframe.F{utils.IndexColName, series.GreaterEq, finishedIdx})

	for idx, row = range filterDf.Maps() {
		if idx < 0xC8 {
			var code [6]byte
			market := byte(row["market"].(int))
			strCode := row["code"].(string)
			copy(code[:], []byte(strCode))
			bonusItem := pkg.StockBonus{market, code}
			stockBonus = append(stockBonus, bonusItem)

			continue
		}

		reqNode := pkg.GenerateStockBonus(stockBonus, 0)
		client.session.Send(reqNode)

		finishedIdx += <- client.bonusFinishedChan

		stockBonus = []pkg.StockBonus{}
		idx = 0
		goto CONTINUE
	}

	if idx > 0 {
		reqNode := pkg.GenerateStockBonus(stockBonus, stockBonusFinishedIdx)
		client.session.Send(reqNode)
	}

	logger.Info("高送转数据接收完毕...")
}

/**
 * 更新股票高送转数据
 */
func (client *TdxClient) UpdateStockBonus(){
	// 股指基
	df := comm.GetFinanceDataFrame(client.Configure, comm.STOCKA, comm.STOCKB, comm.INDEX, comm.FUNDS)
	if nil != df.Err {
		logger.Error("读取股票基础数据失败! err:%v", df)
		return
	}

	client.bonusFinishedChan = make(chan int)
	stockBonus := pkg.GenerateStockBonus(nil, 0)
	client.dispatcher.AddHandler(uint32(stockBonus.EventId), client.OnStockBonus)

	filterDf := df.Filter(dataframe.F{"bonus2", series.Greater, 0})

	client.updateBonus(&filterDf)
}

/**
 * 更新股票日线数据
 */
func (client *TdxClient) UpdateDays(){
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("panic recover! p: %v", p)
		}

		today, _ := strconv.Atoi(utils.Today())

		reqNode := pkg.GenerateStockDayItem(0, "000001", uint32(today), uint32(today), uint16(0xffff)) // index要避免是0，0的话会随机生成idx
		client.session.Send(reqNode)
	}()

	calendar, err := comm.DefaultStockCalendar("")
	if nil != err { logger.Error("UpdateDays Err:%v", err); return }

	// 股指基
	client.stockBaseDF = comm.GetFinanceDataFrame(client.Configure, comm.STOCKA, comm.STOCKB, comm.INDEX, comm.FUNDS)
	if nil != client.stockBaseDF.Err {
		logger.Error("读取股票基础数据失败! err:%v", client.stockBaseDF)
		return
	}

	today, _ := strconv.Atoi(utils.Today())

	dayItem := pkg.GenerateStockDayItem(0, "", 0, 0, 0)
	client.dispatcher.AddHandler(uint32(dayItem.EventId), client.OnStockHistory)

	for idx, row := range client.stockBaseDF.Maps() {
		var code [6]byte
		market := row["market"].(int)
		strCode := row["code"].(string)
		logger.Info("接收 %d%s 的日线数据...", market, strCode)

		copy(code[:], []byte(strCode))

		start := "19901219"
		fileName := fmt.Sprintf("%d%s.csv.zip", market, strCode)

		stocksPath := fmt.Sprintf("%s%s%s", client.Configure.GetApp().DataPath,
			client.Configure.GetTdx().Files.StockDay, fileName)

		colTypes := map[string]series.Type{
			"date": series.Int, "open": series.Float, "low": series.Float, "high": series.Float,
			"close": series.Float, "volume": series.Int, "amount": series.Float}

		stockItemDF := utils.ReadCSV(stocksPath, dataframe.WithTypes(colTypes))

		if nil == stockItemDF.Err {
			// 获取最后一条记录的日期
			idx := utils.FindInStringSlice("date", stockItemDF.Names())
			start, err = calendar.NextDay(stockItemDF.Elem(stockItemDF.Nrow()-1, idx).String())
			if nil != err {
				logger.Error("UpdateDays Err:%v", err)
				return
			}
		}

		tmpStart, _ := strconv.Atoi(start)

		for tmpEnd:=0; tmpEnd < today;  {
			if tmpStart+40000 > today {
				tmpEnd = today
			} else {
				tmpEnd = tmpStart+40000
			}

			reqNode := pkg.GenerateStockDayItem(uint16(market), strCode, uint32(tmpStart), uint32(tmpEnd), uint16(idx+1)) // index要避免是0，0的话会随机生成idx
			client.session.Send(reqNode)

			tmpStart = tmpEnd+1
		}
	}
}

/**
 * 更新股票五分钟线数据
 */
func (client *TdxClient) UpdateMins(){
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("panic recover! p: %v", p)
		}

		today, _ := strconv.Atoi(utils.Today())

		// 通知消费方更新结束
		reqNode := pkg.GenerateStockMinsItem(0, "000001", uint32(today), uint32(today), uint16(0xffff))
		client.session.Send(reqNode)
	}()

	calendar, err := comm.DefaultStockCalendar("")
	if nil != err { logger.Error("UpdateMins Err:%v", err); return }

	// 股指基
	client.stockBaseDF = comm.GetFinanceDataFrame(client.Configure, comm.STOCKA, comm.STOCKB, comm.INDEX, comm.FUNDS)
	if nil != client.stockBaseDF.Err {
		logger.Error("读取股票基础数据失败! err:%v", client.stockBaseDF)
		return
	}

	today, _ := strconv.Atoi(utils.Today())

	minItem := pkg.GenerateStockMinsItem(0, "", 0, 0, 0)
	client.dispatcher.AddHandler(uint32(minItem.EventId), client.OnStockHistory)

	for idx, row := range client.stockBaseDF.Maps() {
		var code [6]byte
		market := row["market"].(int)
		strCode := row["code"].(string)
		logger.Info("接收 %d%s 的五分钟线数据...", market, strCode)

		copy(code[:], []byte(strCode))

		// 默认由今天往前100天
		start := utils.AddDays(utils.Today(), -100)
		fileName := fmt.Sprintf("%d%s.csv.zip", market, strCode)

		stocksPath := fmt.Sprintf("%s%s%s", client.Configure.GetApp().DataPath,
			client.Configure.GetTdx().Files.StockMin, fileName)

		colTypes := map[string]series.Type{
			"date": series.Int, "time": series.String, "open": series.Float, "low": series.Float, "high": series.Float,
			"close": series.Float, "volume": series.Int, "amount": series.Float}

		stockItemDF := utils.ReadCSV(stocksPath, dataframe.WithTypes(colTypes))

		if nil == stockItemDF.Err {
			// 获取最后一条记录的日期
			idx := utils.FindInStringSlice("date", client.stockBaseDF.Names())
			start, err = calendar.NextDay(stockItemDF.Elem(stockItemDF.Nrow()-1, idx).String())
			if nil != err { logger.Error("UpdateMins Err:%v", err); return }
		}

		tmpStart, _ := strconv.Atoi(start)

		for tmpEnd:=0; tmpEnd < today;  {
			resultDate := utils.AddDaysExceptWeekend(fmt.Sprintf("%d", tmpStart), 0x0F)
			nResultDate, _ := strconv.Atoi(resultDate)

			if nResultDate > today { tmpEnd = today } else { tmpEnd = nResultDate }

			reqNode := pkg.GenerateStockMinsItem(uint16(market), strCode, uint32(tmpStart), uint32(tmpEnd), uint16(idx+1))
			client.session.Send(reqNode)

			tmpStart = tmpEnd+1
		}
	}
}
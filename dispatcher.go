package ctdx

import (
	"sync"

	"github.com/datochan/gcom/cnet"

	pkg "github.com/datochan/ctdx/packet"
)

type CTdxDispatcher struct {
	*cnet.Dispatcher
	rwlock     sync.RWMutex    // 写互斥避免并发状态下相互干扰
}

/**
 * 事件分发器
 */
func NewCTdxDispatcher() *CTdxDispatcher {
	return &CTdxDispatcher{Dispatcher: cnet.NewDispatcher()}
}

/**
 * 事件处理过程
 */
func (p *CTdxDispatcher) HandleProc(session *cnet.Session, packet interface{}) {
	p.rwlock.RLock()
	defer p.rwlock.RUnlock()

	respNode := packet.(pkg.ResponseNode)
	if respNode.EventId <= 0 { return }

	handlerProc := p.GetHandler(uint32(respNode.EventId))
	if nil == handlerProc {
		UnknownPkgHandler(session, packet)
		return
	}

	handlerProc(session, packet)
}

package gnet

import (
	"sync"
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/track"

	pkgnet "github.com/panjf2000/gnet/v2"
)

type Request = httpingress.Request

type ConnContext struct {
	PBReq               pb.AdEvent
	TrackReq            track.Request
	Evt                 domain.Event
	ValSlice            []any
	Resp                pb.TrackResponse
	BufSlice            []byte
	ExtraBuf            []byte
	OpenRTBMultiADM     [openrtb.OpenRTB26ImpMax][512]byte
	OpenRTBParsed       openrtb.OpenRTB26Parsed
	ClickParsed         track.ClickQueryParsed
	TelegramClickParsed track.TelegramQueryParsed
	WReqID              filter.BufWrapper
	WCamp               filter.BufWrapper
	WTime               filter.BufWrapper
	RemoteIP            string
	ShardID             int
	WorkerID            int

	OffloadConn     pkgnet.Conn
	OffloadReqBuf   *[]byte
	OffloadReqSlice []byte
	OffloadReqLen   int
	OffloadReq      Request
	OffloadReqPin   bool
	OffloadHTTPPin  []byte

	OffloadArenaWorker int
	OffloadArenaSlot   int
	OffloadRelease     func()

	OffloadOnEnter         func()
	OffloadBlock           <-chan struct{}
	OffloadWG              *sync.WaitGroup
	OffloadAsyncWrite      atomic.Bool
	OffloadCloseAfterWrite atomic.Bool
	OffloadRetired         atomic.Bool
	HTTP1ConnCtx           *ConnContext

	ProtoH2    bool
	H2         httpingress.H2ConnState
	H2StreamID uint32

	HTTP1IncompleteSpin       uint8
	HTTP1BodyIdleArmed        bool
	HTTP1BodyIdleDeadline     int64
	HTTP1ConnOpenedMono       int64
	HTTP1OffloadBusy          atomic.Bool
	HTTP1PendingOffloadWrites atomic.Int32

	ChunkScratch []byte
}

"""Generate internal/ingest/gnet from git internal/ingestion sources."""

from __future__ import annotations

import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "internal/ingest/gnet"


def git_show(path: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(ROOT), "show", f"HEAD:{path}"],
        text=True,
    )


def write(name: str, body: str) -> None:
    (OUT / name).write_text(body)


FIELD_RENAMES = {
    "pbReq": "PBReq",
    "trackReq": "TrackReq",
    "evt": "Evt",
    "valSlice": "ValSlice",
    "resp": "Resp",
    "bufSlice": "BufSlice",
    "extraBuf": "ExtraBuf",
    "openrtbMultiADM": "OpenRTBMultiADM",
    "openrtbParsed": "OpenRTBParsed",
    "clickParsed": "ClickParsed",
    "telegramClickParsed": "TelegramClickParsed",
    "wReqID": "WReqID",
    "wCamp": "WCamp",
    "wTime": "WTime",
    "remoteIP": "RemoteIP",
    "shardID": "ShardID",
    "workerID": "WorkerID",
    "offloadConn": "OffloadConn",
    "offloadReqBuf": "OffloadReqBuf",
    "offloadReqSlice": "OffloadReqSlice",
    "offloadReqLen": "OffloadReqLen",
    "offloadReq": "OffloadReq",
    "offloadReqPin": "OffloadReqPin",
    "offloadHTTPPin": "OffloadHTTPPin",
    "offloadArenaWorker": "OffloadArenaWorker",
    "offloadArenaSlot": "OffloadArenaSlot",
    "offloadRelease": "OffloadRelease",
    "offloadOnEnter": "OffloadOnEnter",
    "offloadBlock": "OffloadBlock",
    "offloadWG": "OffloadWG",
    "offloadAsyncWrite": "OffloadAsyncWrite",
    "offloadCloseAfterWrite": "OffloadCloseAfterWrite",
    "offloadRetired": "OffloadRetired",
    "http1ConnCtx": "HTTP1ConnCtx",
    "protoH2": "ProtoH2",
    "h2": "H2",
    "h2StreamID": "H2StreamID",
    "http1IncompleteSpin": "HTTP1IncompleteSpin",
    "http1BodyIdleArmed": "HTTP1BodyIdleArmed",
    "http1BodyIdleDeadline": "HTTP1BodyIdleDeadline",
    "http1ConnOpenedMono": "HTTP1ConnOpenedMono",
    "http1OffloadBusy": "HTTP1OffloadBusy",
    "http1PendingOffloadWrites": "HTTP1PendingOffloadWrites",
    "chunkScratch": "ChunkScratch",
}


def xform(src: str) -> str:
    src = src.replace("package ingestion", "package gnet")
    src = re.sub(
        r'"github.com/panjf2000/gnet/v2"',
        'pkgnet "github.com/panjf2000/gnet/v2"',
        src,
    )
    src = src.replace("gnet.Conn", "pkgnet.Conn")
    src = src.replace("gnet.Engine", "pkgnet.Engine")
    src = src.replace("gnet.Action", "pkgnet.Action")
    src = src.replace("gnet.None", "pkgnet.None")
    src = src.replace("gnet.Close", "pkgnet.Close")
    src = src.replace("gnet.AsyncCallback", "pkgnet.AsyncCallback")
    src = src.replace("*AdsPacketHandler", "*Server")
    src = src.replace("connContext", "ConnContext")
    src = src.replace("parsedHTTPRequest", "Request")
    for old, new in FIELD_RENAMES.items():
        src = re.sub(r"\b" + old + r"\b", new, src)
    src = src.replace("TrackRequest", "track.Request")
    src = src.replace("bufWrapper", "filter.BufWrapper")
    src = src.replace("h2ConnState", "httpingress.H2ConnState")
    src = src.replace("OpenRTB26Parsed", "openrtb.OpenRTB26Parsed")
    src = src.replace("[openrtb26ImpMax]", "[openrtb.OpenRTB26ImpMax]")
    src = src.replace("clickQueryParsed", "track.ClickQueryParsed")
    src = src.replace("telegramQueryParsed", "track.TelegramQueryParsed")
    src = src.replace("parseHTTP1(", "httpingress.ParseHTTP1(")
    src = src.replace("errIncompleteRequest", "httpingress.ErrIncomplete")
    src = src.replace("errPayloadTooLarge", "httpingress.ErrPayloadTooLarge")
    src = src.replace("isH2ClientPreface", "httpingress.IsH2ClientPreface")
    src = src.replace("h2WrapH1Response", "httpingress.H2WrapH1Response")
    src = src.replace("resetChunkScratch", "httpingress.ResetChunkScratch")
    src = src.replace("pinParsedHTTPRequest", "PinParsedHTTPRequest")
    src = src.replace("monotonicNano()", "filter.MonotonicNano()")
    src = src.replace("monoElapsedSeconds", "filter.MonoElapsedSeconds")
    return src


def ensure_imports(body: str, extra: list[str]) -> str:
    if "import (" not in body:
        return body
    block = "\n".join(f'\t"{imp}"' for imp in extra)
    return body.replace("import (", "import (\n" + block + "\n", 1)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)

    write(
        "doc.go",
        "// Package gnet implements the tracker gnet HTTP ingress engine.\npackage gnet\n",
    )

    write(
        "reactor.go",
        """package gnet

import pkgnet "github.com/panjf2000/gnet/v2"

type Reactor interface {
\tReact(req Request, c pkgnet.Conn) pkgnet.Action
}
""",
    )

    handler = git_show("internal/ingestion/handler.go")
    m = re.search(r"type ConnContext struct \{[^}]+\}", xform(handler), re.DOTALL)
    if not m:
        m = re.search(r"type ConnContext struct \{[^}]+\}", handler, re.DOTALL)
        ctx_struct = xform(m.group(0))
    else:
        ctx_struct = m.group(0)

    write(
        "context.go",
        """package gnet

import (
\t"sync"
\t"sync/atomic"

\t"ad-event-processor/internal/domain"
\t"ad-event-processor/internal/filter"
\t"ad-event-processor/internal/ingest/httpingress"
\t"ad-event-processor/internal/ingest/pb"
\t"ad-event-processor/internal/openrtb"
\t"ad-event-processor/internal/track"

\tpkgnet "github.com/panjf2000/gnet/v2"
)

type Request = httpingress.Request

"""
        + ctx_struct
        + "\n",
    )

    for src_name, dst_name in [
        ("internal/ingestion/worker_pool.go", "worker.go"),
        ("internal/ingestion/worker_arena.go", "arena.go"),
        ("internal/ingestion/gnet_harness.go", "harness.go"),
        ("internal/ingestion/handler_http1_idle.go", "conn_idle.go"),
        ("internal/ingestion/handler_http2.go", "h2.go"),
        ("internal/ingestion/http2_conn.go", "h2_conn.go"),
    ]:
        body = xform(git_show(src_name))
        body = ensure_imports(
            body,
            [
                "ad-event-processor/internal/filter",
                "ad-event-processor/internal/ingest/httpingress",
                "ad-event-processor/internal/track",
                "ad-event-processor/internal/openrtb",
            ],
        )
        write(dst_name, body)

    resp_block = re.search(
        r"respBadRequestClose\s*=\s*\[\]byte.*?\)\n",
        handler,
        re.DOTALL,
    )
    responses = xform(resp_block.group(0)) if resp_block else ""
    write(
        "responses.go",
        "package gnet\n\n" + responses + "\n",
    )

    pools = """package gnet

import "sync"

const maxPoolObjectSize = 64 * 1024

var requestBufferPool = sync.Pool{
\tNew: func() any {
\t\tb := make([]byte, 4096)
\t\treturn &b
\t},
}

func putRequestBuffer(buf *[]byte) {
\tif buf == nil || cap(*buf) > maxPoolObjectSize {
\t\treturn
\t}
\t*buf = (*buf)[:0]
\trequestBufferPool.Put(buf)
}
"""
    write("pools.go", pools)

    server_parts = []
    for pat in [
        r"func \(h \*AdsPacketHandler\) releaseOffloadBuffers.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) retireOffloadContext.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) write\(.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) writeClose.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) writeFilterReject.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) writeMaybeClose.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) OnBoot.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) OnOpen.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) OnClose.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) OnTraffic.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) runOffloadedRequest.*?\n\}\n",
        r"func \(h \*AdsPacketHandler\) parseHTTP.*?\n\}\n",
    ]:
        m = re.search(pat, handler, re.DOTALL)
        if m:
            server_parts.append(xform(m.group(0)))

    server_header = """package gnet

import (
\t"context"
\t"errors"
\t"log/slog"
\t"net/http"
\t"sync"
\t"sync/atomic"

\t"ad-event-processor/internal/config"
\t"ad-event-processor/internal/filter"
\t"ad-event-processor/internal/ingest/httpingress"
\t"ad-event-processor/internal/metrics"
\t"ad-event-processor/pkg/logger"

\tpkgnet "github.com/panjf2000/gnet/v2"
)

type Logger = logger.Logger

type Server struct {
\tpkgnet.BuiltinEventEngine
\teng                *pkgnet.Engine
\tcfg                *config.Config
\treactor            Reactor
\tlogger             *Logger
\tloggerShardCounter atomic.Uint64
\tcontextPool        sync.Pool
\tworkerPool         *PinnedWorkerPool
\tconnWorkerAssign   atomic.Uint64
\tonTrackStatus      func(status int)
\tmonoElapsed        func(start int64) float64
}

type ServerConfig struct {
\tCfg               *config.Config
\tReactor           Reactor
\tLogger            *Logger
\tRecordTrackStatus func(status int)
\tMonoElapsed       func(start int64) float64
}

func NewServer(sc ServerConfig) *Server {
\ts := &Server{
\t\tcfg:           sc.Cfg,
\t\treactor:       sc.Reactor,
\t\tlogger:        sc.Logger,
\t\tonTrackStatus: sc.RecordTrackStatus,
\t\tmonoElapsed:   sc.MonoElapsed,
\t}
\ts.contextPool = sync.Pool{New: func() any { return newConnContext() }}
\treturn s
}

func newConnContext() *ConnContext {
\treturn &ConnContext{
\t\tPBReq: pb.AdEvent{Metadata: &pb.EventMetadata{}},
\t\tTrackReq: track.Request{Payload: make([]byte, 0, 512)},
\t\tEvt: domain.Event{Payload: make([]byte, 0, 1024)},
\t\tValSlice: make([]any, 18),
\t\tBufSlice: make([]byte, 4096),
\t\tExtraBuf: make([]byte, 0, 4096),
\t\tOffloadHTTPPin: make([]byte, 0, 2048),
\t\tChunkScratch: make([]byte, 0, 4096),
\t\tH2: httpingress.NewH2ConnState(),
\t\tWReqID: filter.BufWrapper{Buf: make([]byte, 0, 128)},
\t\tWCamp: filter.BufWrapper{Buf: make([]byte, 0, 128)},
\t\tWTime: filter.BufWrapper{Buf: make([]byte, 0, 128)},
\t\tWorkerID: -1,
\t}
}

func (s *Server) SetReactor(r Reactor) { s.reactor = r }
func (s *Server) SetLogger(l *Logger) { s.logger = l }
func (s *Server) SetWorkerPool(wp *PinnedWorkerPool) {
\ts.workerPool = wp
\tif wp != nil {
\t\twp.handler = s
\t}
}

func (s *Server) React(req Request, c pkgnet.Conn) pkgnet.Action {
\tif s.reactor == nil {
\t\treturn pkgnet.None
\t}
\treturn s.reactor.React(req, c)
}

func (s *Server) Stop(ctx context.Context) error {
\tif s.eng != nil {
\t\treturn s.eng.Stop(ctx)
\t}
\treturn nil
}

func (s *Server) Write(c pkgnet.Conn, data []byte, ctx *ConnContext) { s.write(c, data, ctx) }
func (s *Server) WriteClose(c pkgnet.Conn, data []byte, ctx *ConnContext) { s.writeClose(c, data, ctx) }
func (s *Server) WriteFilterReject(c pkgnet.Conn, data []byte, ctx *ConnContext, closeDup bool) {
\tif closeDup {
\t\ts.writeClose(c, data, ctx)
\t\treturn
\t}
\ts.write(c, data, ctx)
}
func (s *Server) AllocConnContext(c pkgnet.Conn) *ConnContext { return s.allocConnContext(c) }

func (s *Server) recordTrackStatus(status int) {
\tif s.onTrackStatus != nil {
\t\ts.onTrackStatus(status)
\t}
}

"""
    server_body = server_header + "\n".join(
        p.replace("(h *Server)", "(s *Server)").replace(" h.", " s.").replace("\th.", "\ts.")
        for p in server_parts
    )
    server_body = ensure_imports(
        server_body,
        [
            "ad-event-processor/internal/domain",
            "ad-event-processor/internal/ingest/pb",
            "ad-event-processor/internal/track",
        ],
    )
    write("server.go", server_body)

    harness_extra = git_show("internal/ingestion/gnet_harness.go")
    for fn in [
        "BuildGnetHTTP",
        "BuildGnetPostTrackJSON",
        "BuildGnetGetHealth",
        "BuildGnetGetReady",
        "GetHealthGnet",
        "GetReadyGnet",
        "ParseGnetHTTPStatus",
        "ParseGnetHTTPBody",
        "ServeGnetHarness",
    ]:
        m = re.search(rf"func {fn}\(.*?\n\}}\n", harness_extra + handler, re.DOTALL)
        if m:
            part = xform(m.group(0))
            path = OUT / "harness_export.go"
            cur = path.read_text() if path.exists() else "package gnet\n\nimport (\n\t\"bytes\"\n\t\"strconv\"\n\n\t\"ad-event-processor/internal/ingest/httpingress\"\n\n\tpkgnet \"github.com/panjf2000/gnet/v2\"\n)\n\n"
            path.write_text(cur + part + "\n")

    print(f"wrote gnet package to {OUT}")


if __name__ == "__main__":
    main()

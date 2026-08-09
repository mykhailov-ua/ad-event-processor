package ingestion

import "errors"

// http1TrackEdgePolicy mirrors deploy/nginx/lua/edge-phase2.lua wire policy for POST /track:
// Content-Length is required (no chunked transfer on the edge path).
func http1TrackEdgePolicy(req *parsedHTTPRequest, hFlags uint8) error {
	if req == nil || !isPOSTTrack(req) {
		return nil
	}
	if hFlags&http1flChunkedTE != 0 {
		return errInvalidRequest
	}
	if hFlags&http1flCLSet == 0 {
		return errInvalidRequest
	}
	return nil
}

func isPOSTTrack(req *parsedHTTPRequest) bool {
	return len(req.Method) == 4 &&
		req.Method[0] == 'P' && req.Method[1] == 'O' && req.Method[2] == 'S' && req.Method[3] == 'T' &&
		httpPathHasPrefix(req.Path, "/track")
}

// IngressVerdict is the normalized accept/reject outcome for cross-hop parity tests.
type IngressVerdict string

const (
	IngressAccept     IngressVerdict = "accept"
	IngressReject     IngressVerdict = "reject"
	IngressIncomplete IngressVerdict = "incomplete"
)

type ingressDisposition struct {
	Verdict  IngressVerdict
	BodyLen  int
	Consumed int
}

func dispositionFromHTTP1Parse(n int, req parsedHTTPRequest, err error) ingressDisposition {
	if err != nil {
		if errors.Is(err, errIncompleteRequest) {
			return ingressDisposition{Verdict: IngressIncomplete}
		}
		return ingressDisposition{Verdict: IngressReject}
	}
	bodyLen := req.ContentLength
	if req.Body != nil {
		bodyLen = len(req.Body)
	}
	return ingressDisposition{
		Verdict:  IngressAccept,
		BodyLen:  bodyLen,
		Consumed: n,
	}
}

// edgeHTTP1Disposition models nginx edge-phase2 + tracker parse for /track ingress.
func edgeHTTP1Disposition(wire []byte, maxBody int64) ingressDisposition {
	return gnetHTTP1Disposition(wire, maxBody)
}

// gnetHTTP1Disposition is the tracker gnet parse outcome.
func gnetHTTP1Disposition(wire []byte, maxBody int64) ingressDisposition {
	n, req, err := parseHTTP1(wire, maxBody, nil)
	return dispositionFromHTTP1Parse(n, req, err)
}

// http1IngressCanonical compares edge and gnet dispositions (must match after shared policy).
func http1IngressCanonical(wire []byte, maxBody int64) (edge, gnet ingressDisposition, differential bool) {
	edge = edgeHTTP1Disposition(wire, maxBody)
	gnet = gnetHTTP1Disposition(wire, maxBody)
	differential = edge.Verdict != gnet.Verdict ||
		(edge.Verdict == IngressAccept && edge.BodyLen != gnet.BodyLen) ||
		(edge.Verdict == IngressAccept && edge.Consumed != gnet.Consumed)
	return edge, gnet, differential
}

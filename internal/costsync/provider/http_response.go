package provider

import (
	"io"
	"net/http"

	"ad-event-processor/pkg/coldpath"
)

func doReadLimitedHTTPBody(client *http.Client, req *http.Request, limit int64) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func readLimitedHTTPBody(resp *http.Response, limit int64) ([]byte, error) {
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, err
	}
	return body, nil
}

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// CreateFile uploads encrypted binary file data to the streaming endpoint and
// returns the file ID. The filename is embedded in the OpenPGP payload by the
// caller, never sent in cleartext.
func (c *Client) CreateFile(ctx context.Context, r io.Reader, expiration int32, oneTime bool) (string, error) {
	headers := map[string]string{
		"Content-Type":        "application/octet-stream",
		"X-Yopass-Expiration": strconv.FormatInt(int64(expiration), 10),
		"X-Yopass-OneTime":    strconv.FormatBool(oneTime),
	}
	return c.message(ctx, http.MethodPost, "/create/file", r, headers)
}

// FetchFile retrieves an encrypted file by ID. The caller owns the returned
// body and must close it. size is the Content-Length (-1 if unknown).
func (c *Client) FetchFile(ctx context.Context, id string) (body io.ReadCloser, size int64, err error) {
	resp, err := c.do(ctx, http.MethodGet, "/file/"+id, nil,
		map[string]string{"Accept": "application/octet-stream"})
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, 0, serverError(resp.StatusCode, raw)
	}
	return resp.Body, resp.ContentLength, nil
}

// FetchFileBytes is a convenience wrapper that reads the whole file into
// memory. Prefer FetchFile for large downloads with progress reporting.
func (c *Client) FetchFileBytes(ctx context.Context, id string) ([]byte, error) {
	body, _, err := c.FetchFile(ctx, id)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read file body: %w", err)
	}
	return data, nil
}

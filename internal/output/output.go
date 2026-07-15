// Package output renders command results either as human-readable text or as
// machine-readable JSON, so the same commands serve interactive users and CI.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// SendResult is the outcome of a send command.
type SendResult struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Key        string `json:"key"`
	ManualKey  bool   `json:"manual_key"`
	File       bool   `json:"file"`
	OneTime    bool   `json:"one_time"`
	Expiration string `json:"expiration"`
}

// ReceiveResult is the outcome of a receive command. For a text secret,
// Content holds the plaintext; for a file, Written/Filename/Bytes describe the
// output written to disk.
type ReceiveResult struct {
	Content  string `json:"content,omitempty"`
	Filename string `json:"filename,omitempty"`
	Written  string `json:"written,omitempty"`
	Bytes    int    `json:"bytes,omitempty"`
}

// Printer renders results to output streams.
type Printer interface {
	Send(SendResult) error
	Receive(ReceiveResult) error
	Error(code int, message string) error
}

// New returns a JSON printer when jsonMode is set, otherwise a text printer.
// out receives success payloads; errw receives diagnostics.
func New(jsonMode bool, out, errw io.Writer) Printer {
	if jsonMode {
		return &jsonPrinter{out: out, errw: errw}
	}
	return &textPrinter{out: out, errw: errw}
}

type textPrinter struct{ out, errw io.Writer }

func (p *textPrinter) Send(r SendResult) error {
	if _, err := fmt.Fprintln(p.out, r.URL); err != nil {
		return err
	}
	// In manual-key mode the key is not in the URL; surface it on stderr so
	// stdout stays a clean, pipeable URL.
	if r.ManualKey && r.Key != "" {
		_, err := fmt.Fprintf(p.errw, "key: %s\n", r.Key)
		return err
	}
	return nil
}

func (p *textPrinter) Receive(r ReceiveResult) error {
	if r.Written != "" {
		_, err := fmt.Fprintf(p.errw, "wrote %s (%d bytes)\n", r.Written, r.Bytes)
		return err
	}
	_, err := fmt.Fprint(p.out, r.Content)
	return err
}

func (p *textPrinter) Error(_ int, message string) error {
	_, err := fmt.Fprintln(p.errw, "error:", message)
	return err
}

type jsonPrinter struct{ out, errw io.Writer }

func (p *jsonPrinter) Send(r SendResult) error       { return encode(p.out, r) }
func (p *jsonPrinter) Receive(r ReceiveResult) error { return encode(p.out, r) }

func (p *jsonPrinter) Error(code int, message string) error {
	return encode(p.errw, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func encode(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

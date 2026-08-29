// Package mcp speaks the Model Context Protocol on Orbit's behalf, so that a
// supervising model can read the board and act on it through the very
// functions the command line and the window call.
//
// The server holds no authority of its own. Every tool goes through
// internal/task and internal/board, which is what keeps a model's mistake to
// the same blast radius as a reader's: it can write a task down, start one
// that is not running, leave a note and ask a run to stop, and it reaches
// none of those through a path it built itself.
package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// protocolVersion is the revision of the Model Context Protocol this server
// implements. It is answered as a constant rather than echoed back from the
// client's request: echoing agrees to a revision this code has never seen,
// and a client that cannot work with this one is entitled to say so.
const protocolVersion = "2024-11-05"

// maxLine is the largest single JSON-RPC message this server will read.
//
// A tool call carrying a corrective prompt is the big one, and it is
// kilobytes. Ten megabytes is far past anything a client sends and far short
// of a line that would exhaust memory, which is what bufio.Scanner's default
// 64 KB would refuse and what no limit at all would allow.
const maxLine = 10 * 1024 * 1024

// Server reads JSON-RPC requests, one per line, and writes one response per
// request that has an id.
type Server struct {
	in      io.Reader
	out     io.Writer
	session Session
}

// NewServer builds a server that reads from in, writes to out, and answers
// every tool call against the given session.
//
// out must be the stream the client is reading, and nothing else may write
// to it. That is the one thing this transport asks of its host: a single
// stray line of logging on the same descriptor is a parse error at the
// client and a session that ends. Orbit's logger writes to a file under the
// state root for exactly this reason.
func NewServer(in io.Reader, out io.Writer, session Session) *Server {
	return &Server{in: in, out: out, session: session}
}

// Serve answers requests until the input ends, which is how a client says it
// is done: it closes the pipe.
func (s *Server) Serve() error {
	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLine)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			noteFault("a line from the client would not parse", err)
			// The id is unknown — that is what failed to parse — so the
			// response carries a null one, which is what the specification
			// says a parse error answers with.
			if err := s.send(JSONRPCResponse{JSONRPC: "2.0", Error: &JSONRPCError{Code: CodeParseError, Message: err.Error()}}); err != nil {
				return err
			}

			continue
		}

		if err := s.answer(req); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read from the client: %w", err)
	}

	return nil
}

// answer handles one request. A request with no id is a notification, and a
// notification is never replied to — not even to refuse it, which is the
// rule a client enforces by treating an unsolicited response as a fault.
func (s *Server) answer(req JSONRPCRequest) error {
	if req.ID == nil {
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.send(result(req.ID, InitializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities:    ServerCaps{Tools: &ToolsCap{}},
			ServerInfo:      ServerInfo{Name: "orbit", Version: s.session.Version},
			Instructions:    instructions,
		}))
	case "tools/list":
		return s.send(result(req.ID, map[string]any{"tools": Tools()}))
	case "tools/call":
		return s.call(req)
	case "ping":
		return s.send(result(req.ID, map[string]any{}))
	default:
		why := fmt.Sprintf("this server has no method %q", req.Method)
		noteFault("a method nobody implements", errors.New(why))

		return s.send(fault(req.ID, CodeMethodNotFound, why))
	}
}

// call runs one tool.
//
// Params that will not decode are a transport fault and come back as a
// JSON-RPC error. What the tool itself says — including "there is no task by
// that id" — comes back as a successful result carrying isError, because
// that is the answer the model has to read and act on rather than one the
// client swallows.
func (s *Server) call(req JSONRPCRequest) error {
	var params CallToolParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			noteFault("the parameters of a tools/call would not decode", err)

			return s.send(fault(req.ID, CodeInvalidParams, err.Error()))
		}
	}

	if params.Name == "" {
		noteFault("a tools/call", errors.New("no tool was named"))

		return s.send(fault(req.ID, CodeInvalidParams, "tools/call needs a tool name"))
	}

	return s.send(result(req.ID, s.session.Call(params.Name, params.Arguments)))
}

// result and fault build the two shapes a response can have.
func result(id, payload any) JSONRPCResponse {
	return JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: payload}
}

func fault(id any, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: message}}
}

// send writes one response and the newline that ends it.
//
// A response that cannot be encoded or cannot be written ends the session. It
// used to be dropped: a client that asked a question and got silence waits
// for ever, which is a worse failure than a server that exits and can be
// restarted.
func (s *Server) send(resp JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode a response to request %v: %w", resp.ID, err)
	}

	if _, err := fmt.Fprintf(s.out, "%s\n", data); err != nil {
		return fmt.Errorf("write a response to the client: %w", err)
	}

	return nil
}

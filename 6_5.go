package h2spec

import (
	"fmt"
	"github.com/bradfitz/http2"
	"io"
	"net"
	"syscall"
)

func SettingsTestGroup() *TestGroup {
	tg := NewTestGroup("6.5", "SETTINGS")

	tg.AddTestCase(NewTestCase(
		"Sends a SETTINGS frame",
		"The endpoint MUST sends a SETTINGS frame with ACK.",
		func(ctx *Context) (expected []Result, actual Result) {
			expected = []Result{
				&ResultFrame{http2.FrameSettings, http2.FlagSettingsAck, ErrCodeDefault},
			}

			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			settings := http2.Setting{http2.SettingMaxConcurrentStreams, 100}
			http2Conn.fr.WriteSettings(settings)

		loop:
			for {
				f, err := http2Conn.ReadFrame(ctx.Timeout)
				if err != nil {
					opErr, ok := err.(*net.OpError)
					if err == io.EOF || (ok && opErr.Err == syscall.ECONNRESET) {
						actual = &ResultConnectionClose{}
					} else if err == TIMEOUT {
						if actual == nil {
							actual = &ResultTestTimeout{}
						}
					} else {
						actual = &ResultError{err}
					}
					break loop
				}
				switch f := f.(type) {
				case *http2.SettingsFrame:
					actual = &ResultFrame{f.Header().Type, f.Header().Flags, ErrCodeDefault}
					if f.IsAck() {
						break loop
					}
				default:
					actual = &ResultFrame{f.Header().Type, FlagDefault, ErrCodeDefault}
				}
			}

			return expected, actual
		},
	))

	tg.AddTestCase(NewTestCase(
		"Sends a SETTINGS frame that is not a zero-length with ACK flag",
		"The endpoint MUST respond with a connection error of type FRAME_SIZE_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x01\x04\x01\x00\x00\x00\x00\x00")

			actualCodes := []http2.ErrCode{http2.ErrCodeFrameSize}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestCase(NewTestCase(
		"Sends a SETTINGS frame with the stream identifier that is not 0x0",
		"The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x06\x04\x00\x00\x00\x00\x03")
			fmt.Fprintf(http2Conn.conn, "\x00\x03\x00\x00\x00\x64")

			actualCodes := []http2.ErrCode{http2.ErrCodeProtocol}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestCase(NewTestCase(
		"Sends a SETTINGS frame with a length other than a multiple of 6 octets",
		"The endpoint MUST respond with a connection error of type FRAME_SIZE_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x02\x04\x00\x00\x00\x00\x00")
			fmt.Fprintf(http2Conn.conn, "\x00\x00\x01")

			actualCodes := []http2.ErrCode{http2.ErrCodeFrameSize}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestGroup(DefinedSettingsParametersTestGroup())

	return tg
}

func DefinedSettingsParametersTestGroup() *TestGroup {
	tg := NewTestGroup("6.5.2", "Defined SETTINGS Parameters")

	tg.AddTestCase(NewTestCase(
		"SETTINGS_ENABLE_PUSH (0x2): Sends the value other than 0 or 1",
		"The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x06\x04\x00\x00\x00\x00\x00")
			fmt.Fprintf(http2Conn.conn, "\x00\x02\x00\x00\x00\x02")

			actualCodes := []http2.ErrCode{http2.ErrCodeProtocol}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestCase(NewTestCase(
		"SETTINGS_INITIAL_WINDOW_SIZE (0x4): Sends the value above the maximum flow control window size",
		"The endpoint MUST respond with a connection error of type FLOW_CONTROL_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x06\x04\x00\x00\x00\x00\x00")
			fmt.Fprintf(http2Conn.conn, "\x00\x04\x80\x00\x00\x00")

			actualCodes := []http2.ErrCode{http2.ErrCodeFlowControl}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestCase(NewTestCase(
		"SETTINGS_MAX_FRAME_SIZE (0x5): Sends the value below the initial value",
		"The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x06\x04\x00\x00\x00\x00\x00")
			fmt.Fprintf(http2Conn.conn, "\x00\x05\x00\x00\x3f\xff")

			actualCodes := []http2.ErrCode{http2.ErrCodeProtocol}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	tg.AddTestCase(NewTestCase(
		"SETTINGS_MAX_FRAME_SIZE (0x5): Sends the value above the maximum allowed frame size",
		"The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		func(ctx *Context) (expected []Result, actual Result) {
			http2Conn := CreateHttp2Conn(ctx, true)
			defer http2Conn.conn.Close()

			fmt.Fprintf(http2Conn.conn, "\x00\x00\x06\x04\x00\x00\x00\x00\x00")
			fmt.Fprintf(http2Conn.conn, "\x00\x05\x01\x00\x00\x00")

			actualCodes := []http2.ErrCode{http2.ErrCodeProtocol}
			return TestConnectionError(ctx, http2Conn, actualCodes)
		},
	))

	return tg
}

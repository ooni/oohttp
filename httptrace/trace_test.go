// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httptrace

import (
	"bytes"
	"context"
	"testing"
)

func TestWithClientTrace(t *testing.T) {
	var buf bytes.Buffer
	connectStart := func(b byte) func(network, addr string) {
		return func(network, addr string) {
			buf.WriteByte(b)
		}
	}

	ctx := context.Background()
	oldtrace := &ClientTrace{
		ConnectStart: connectStart('O'),
	}
	ctx = WithClientTrace(ctx, oldtrace)
	newtrace := &ClientTrace{
		ConnectStart: connectStart('N'),
	}
	ctx = WithClientTrace(ctx, newtrace)
	trace := ContextClientTrace(ctx)

	buf.Reset()
	trace.ConnectStart("net", "addr")
	if got, want := buf.String(), "NO"; got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestCompose(t *testing.T) {
	var buf bytes.Buffer
	var testNum int

	connectStart := func(b byte) func(network, addr string) {
		return func(network, addr string) {
			if addr != "addr" {
				t.Errorf(`%d. args for %q case = %q, %q; want addr of "addr"`, testNum, b, network, addr)
			}
			buf.WriteByte(b)
		}
	}

	tests := [...]struct {
		trace, old *ClientTrace
		want       string
	}{
		0: {
			want: "T",
			trace: &ClientTrace{
				ConnectStart: connectStart('T'),
			},
		},
		1: {
			want: "TO",
			trace: &ClientTrace{
				ConnectStart: connectStart('T'),
			},
			old: &ClientTrace{ConnectStart: connectStart('O')},
		},
		2: {
			want:  "O",
			trace: &ClientTrace{},
			old:   &ClientTrace{ConnectStart: connectStart('O')},
		},
	}
	for i, tt := range tests {
		testNum = i
		buf.Reset()

		tr := *tt.trace
		tr.compose(tt.old)
		if tr.ConnectStart != nil {
			tr.ConnectStart("net", "addr")
		}
		if got := buf.String(); got != tt.want {
			t.Errorf("%d. got = %q; want %q", i, got, tt.want)
		}
	}

}

func TestClientTraceWithoutComposition(t *testing.T) {
	unexpectedFlag := false
	oldTrace := &ClientTrace{
		GetConn: func(hostPort string) {
			unexpectedFlag = true
		},
	}
	ctx := context.Background()
	ctx = WithClientTraceWithoutComposition(ctx, oldTrace)
	expectedFlag := false
	newTrace := &ClientTrace{
		GetConn: func(hostPort string) {
			expectedFlag = true
		},
	}
	ctx = WithClientTraceWithoutComposition(ctx, newTrace)
	maybeCompoundTrace := ContextClientTrace(ctx)
	maybeCompoundTrace.GetConn("1.2.3.4")
	if unexpectedFlag {
		t.Fatal("should be false")
	}
	if !expectedFlag {
		t.Fatal("should be true")
	}
}

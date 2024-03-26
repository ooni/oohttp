// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix || (js && wasm) || wasip1

package safefilepath

import "github.com/ooni/oohttp/internal/bytealg"

func localize(path string) (string, error) {
	if bytealg.IndexByteString(path, 0) >= 0 {
		return "", errInvalidPath
	}
	return path, nil
}

// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package godebug makes the settings in the $GODEBUG environment variable
// available to other packages. These settings are often used for compatibility
// tweaks, when we need to change a default behavior but want to let users
// opt back in to the original. For example GODEBUG=http2server=0 disables
// HTTP/2 support in the net/http server.
//
// In typical usage, code should declare a Setting as a global
// and then call Value each time the current setting value is needed:
//
//	var http2server = godebug.New("http2server")
//
//	func ServeConn(c net.Conn) {
//		if http2server.Value() == "0" {
//			disallow HTTP/2
//			...
//		}
//		...
//	}
//
// Each time a non-default setting causes a change in program behavior,
// code should call [Setting.IncNonDefault] to increment a counter that can
// be reported by [runtime/metrics.Read].
// Note that counters used with IncNonDefault must be added to
// various tables in other packages. See the [Setting.IncNonDefault]
// documentation for details.
package godebug

// Note: Be careful about new imports here. Any package
// that internal/godebug imports cannot itself import internal/godebug,
// meaning it cannot introduce a GODEBUG setting of its own.
// We keep imports to the absolute bare minimum.
import (
	_ "runtime"
	"sync"
	"sync/atomic"
	"unsafe"
	_ "unsafe" // go:linkname

	"github.com/ooni/oohttp/internal/bisect"
	"github.com/ooni/oohttp/internal/godebugs"
)

// A Setting is a single setting in the $GODEBUG environment variable.
type Setting struct {
	name string
	once sync.Once
	*setting
}

type setting struct {
	value          atomic.Pointer[value]
	nonDefaultOnce sync.Once
	nonDefault     atomic.Uint64
	info           *godebugs.Info
}

type value struct {
	text   string
	bisect *bisect.Matcher
}

// New returns a new Setting for the $GODEBUG setting with the given name.
//
// GODEBUGs meant for use by end users must be listed in ../godebugs/table.go,
// which is used for generating and checking various documentation.
// If the name is not listed in that table, New will succeed but calling Value
// on the returned Setting will panic.
// To disable that panic for access to an undocumented setting,
// prefix the name with a #, as in godebug.New("#gofsystrace").
// The # is a signal to New but not part of the key used in $GODEBUG.
func New(name string) *Setting {
	return &Setting{name: name}
}

// Name returns the name of the setting.
func (s *Setting) Name() string {
	if s.name != "" && s.name[0] == '#' {
		return s.name[1:]
	}
	return s.name
}

// Undocumented reports whether this is an undocumented setting.
func (s *Setting) Undocumented() bool {
	return s.name != "" && s.name[0] == '#'
}

// String returns a printable form for the setting: name=value.
func (s *Setting) String() string {
	return s.Name() + "=" + s.Value()
}

// IncNonDefault increments the non-default behavior counter
// associated with the given setting.
// This counter is exposed in the runtime/metrics value
// /godebug/non-default-behavior/<name>:events.
//
// Note that Value must be called at least once before IncNonDefault.
func (s *Setting) IncNonDefault() {
	s.nonDefaultOnce.Do(s.register)
	s.nonDefault.Add(1)
}

func (s *Setting) register() {
	if s.info == nil || s.info.Opaque {
		panic("godebug: unexpected IncNonDefault of " + s.name)
	}
	registerMetric("/godebug/non-default-behavior/"+s.Name()+":events", s.nonDefault.Load)
}

// cache is a cache of all the GODEBUG settings,
// a locked map[string]*atomic.Pointer[string].
//
// All Settings with the same name share a single
// *atomic.Pointer[string], so that when GODEBUG
// changes only that single atomic string pointer
// needs to be updated.
//
// A name appears in the values map either if it is the
// name of a Setting for which Value has been called
// at least once, or if the name has ever appeared in
// a name=value pair in the $GODEBUG environment variable.
// Once entered into the map, the name is never removed.
var cache sync.Map // name string -> value *atomic.Pointer[string]

var empty value

// Value returns the current value for the GODEBUG setting s.
//
// Value maintains an internal cache that is synchronized
// with changes to the $GODEBUG environment variable,
// making Value efficient to call as frequently as needed.
// Clients should therefore typically not attempt their own
// caching of Value's result.
func (s *Setting) Value() string {
	s.once.Do(func() {
		s.setting = lookup(s.Name())
		if s.info == nil && !s.Undocumented() {
			panic("godebug: Value of name not listed in godebugs.All: " + s.name)
		}
	})
	v := *s.value.Load()
	if v.bisect != nil && !v.bisect.Stack(&stderr) {
		return ""
	}
	return v.text
}

// lookup returns the unique *setting value for the given name.
func lookup(name string) *setting {
	if v, ok := cache.Load(name); ok {
		return v.(*setting)
	}
	s := new(setting)
	s.info = godebugs.Lookup(name)
	s.value.Store(&empty)
	if v, loaded := cache.LoadOrStore(name, s); loaded {
		// Lost race: someone else created it. Use theirs.
		return v.(*setting)
	}

	return s
}

var godebugDefault string
var godebugUpdate atomic.Pointer[func(string, string)]
var godebugEnv atomic.Pointer[string] // set by parsedebugvars
var godebugNewIncNonDefault atomic.Pointer[func(string) func()]

// setUpdate mimcs runtime.godebug_setUpdate
func setUpdate(update func(string, string)) {
	p := new(func(string, string))
	*p = update
	godebugUpdate.Store(p)
	godebugNotify(false)
}

// godebugNotify mimics runtime.godebugNotify
func godebugNotify(envChanged bool) {
	update := godebugUpdate.Load()
	var env string
	if p := godebugEnv.Load(); p != nil {
		env = *p
	}
	// NOTE: code specific to github.com/ooni/oohttp
	// we omit this check since the only invocation we have is with envChanged = false
	// this is done to avoid forking a large portion of the runtime and internal packages
	// if envChanged {
	// reparsedebugvars(env)
	// }
	if update != nil {
		(*update)(godebugDefault, env)
	}
}

// registerMetric is an empty function
func registerMetric(name string, read func() uint64) {
	// we do not register metrics for this godebug fork
}

// setNewIncNonDefault mimics runtime.godebug_setNewIncNonDefault
func setNewIncNonDefault(newIncNonDefault func(string) func()) {
	p := new(func(string) func())
	*p = newIncNonDefault
	godebugNewIncNonDefault.Store(p)
}

func init() {
	setUpdate(update)
	setNewIncNonDefault(newIncNonDefault)
}

func newIncNonDefault(name string) func() {
	s := New(name)
	s.Value()
	return s.IncNonDefault
}

var updateMu sync.Mutex

// update records an updated GODEBUG setting.
// def is the default GODEBUG setting for the running binary,
// and env is the current value of the $GODEBUG environment variable.
func update(def, env string) {
	updateMu.Lock()
	defer updateMu.Unlock()

	// Update all the cached values, creating new ones as needed.
	// We parse the environment variable first, so that any settings it has
	// are already locked in place (did[name] = true) before we consider
	// the defaults.
	did := make(map[string]bool)
	parse(did, env)
	parse(did, def)

	// Clear any cached values that are no longer present.
	cache.Range(func(name, s any) bool {
		if !did[name.(string)] {
			s.(*setting).value.Store(&empty)
		}
		return true
	})
}

// parse parses the GODEBUG setting string s,
// which has the form k=v,k2=v2,k3=v3.
// Later settings override earlier ones.
// Parse only updates settings k=v for which did[k] = false.
// It also sets did[k] = true for settings that it updates.
// Each value v can also have the form v#pattern,
// in which case the GODEBUG is only enabled for call stacks
// matching pattern, for use with golang.org/x/tools/cmd/bisect.
func parse(did map[string]bool, s string) {
	// Scan the string backward so that later settings are used
	// and earlier settings are ignored.
	// Note that a forward scan would cause cached values
	// to temporarily use the ignored value before being
	// updated to the "correct" one.
	end := len(s)
	eq := -1
	for i := end - 1; i >= -1; i-- {
		if i == -1 || s[i] == ',' {
			if eq >= 0 {
				name, arg := s[i+1:eq], s[eq+1:end]
				if !did[name] {
					did[name] = true
					v := &value{text: arg}
					for j := 0; j < len(arg); j++ {
						if arg[j] == '#' {
							v.text = arg[:j]
							v.bisect, _ = bisect.New(arg[j+1:])
							break
						}
					}
					lookup(name).value.Store(v)
				}
			}
			eq = -1
			end = i
		} else if s[i] == '=' {
			eq = i
		}
	}
}

type runtimeStderr struct{}

var stderr runtimeStderr

func (*runtimeStderr) Write(b []byte) (int, error) {
	if len(b) > 0 {
		write(2, unsafe.Pointer(&b[0]), int32(len(b)))
	}
	return len(b), nil
}

// Since we cannot import os or syscall, use the runtime's write function
// to print to standard error.
//
//go:linkname write runtime.write
func write(fd uintptr, p unsafe.Pointer, n int32) int32

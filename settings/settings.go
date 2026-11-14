// Package settings keeps small values that outlive the app - settings,
// progress, the last opened file - as JSON in the app's data directory
// (the browser's local storage on the web). It's for plain data: keep
// secrets in giode.SecureStore.
//
//	settings.Save("theme", "dark")
//	theme := settings.Value("theme", "light")
//
//	// A State that loads on start and saves itself on every change:
//	progress := settings.Bind("progress", map[string]bool{})
//	progress.Update(func(p map[string]bool) map[string]bool { p["orbits"] = true; return p })
package settings

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/xmhrnx/giode/state"
)

// Store is a set of named values kept as one JSON document. Its methods
// may be called from any goroutine. Changes are written at once, the
// whole document replaced atomically, so a crash never leaves it half
// written.
type Store struct {
	mu      sync.Mutex
	backend backend
	values  map[string]json.RawMessage
	loaded  bool
	err     error
}

// backend reads and writes the document.
type backend interface {
	load() ([]byte, error)
	save([]byte) error
}

// ErrNoStorage is returned when the platform has nowhere to keep
// settings.
var ErrNoStorage = errors.New("settings: no storage on this platform")

// Default is the app's store: settings.json in the app's data directory
// (its own folder under the user's config directory on desktops, the
// app's private files on Android and iOS), or the browser's local
// storage on the web.
var Default = &Store{backend: defaultBackend()}

// Open returns a store kept in the file at path.
func Open(path string) *Store {
	return &Store{backend: fileBackend{path: path}}
}

// load reads the document once. The caller holds s.mu.
func (s *Store) load() error {
	if s.loaded {
		return s.err
	}
	s.loaded = true
	s.values = map[string]json.RawMessage{}
	if s.backend == nil {
		s.err = ErrNoStorage
		return s.err
	}
	b, err := s.backend.load()
	if err != nil {
		s.err = err
		return err
	}
	if len(b) > 0 {
		// A damaged document is dropped rather than blocking the app.
		if json.Unmarshal(b, &s.values) != nil {
			s.values = map[string]json.RawMessage{}
		}
	}
	return nil
}

// Get decodes the value stored under key into v, a pointer, and reports
// whether there was one that decoded.
func (s *Store) Get(key string, v any) bool {
	s.mu.Lock()
	s.load()
	raw, ok := s.values[key]
	s.mu.Unlock()
	return ok && json.Unmarshal(raw, v) == nil
}

// Has reports whether a value is stored under key.
func (s *Store) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	_, ok := s.values[key]
	return ok
}

// Set stores v, anything encoding/json encodes, under key and writes
// the store.
func (s *Store) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// A store that couldn't be read isn't written over: that would lose
	// what it holds.
	if err := s.load(); err != nil {
		return err
	}
	s.values[key] = raw
	return s.write()
}

// Delete removes key and writes the store; a missing key is not an
// error.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	if _, ok := s.values[key]; !ok {
		return nil
	}
	delete(s.values, key)
	return s.write()
}

// Keys returns the stored keys, sorted.
func (s *Store) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	return slices.Sorted(maps.Keys(s.values))
}

// write saves the document. The caller holds s.mu.
func (s *Store) write() error {
	if s.backend == nil {
		return ErrNoStorage
	}
	b, err := json.MarshalIndent(s.values, "", "\t")
	if err != nil {
		return err
	}
	if err := s.backend.save(b); err != nil {
		return err
	}
	s.err = nil
	return nil
}

// Value returns the value stored under key in the Default store, or def
// when there's none (or it doesn't decode as a T).
func Value[T any](key string, def T) T {
	var v T
	if Default.Get(key, &v) {
		return v
	}
	return def
}

// Save stores v under key in the Default store.
func Save(key string, v any) error {
	return Default.Set(key, v)
}

// Delete removes key from the Default store.
func Delete(key string) error {
	return Default.Delete(key)
}

// Bind returns a State holding the value saved under key in the Default
// store (initial when there's none) that saves itself every time it
// changes. Make it once, as a package-level var or beside the
// component that uses it.
func Bind[T any](key string, initial T) *state.State[T] {
	return BindTo(Default, key, initial)
}

// BindTo is Bind for another store.
func BindTo[T any](s *Store, key string, initial T) *state.State[T] {
	v := initial
	var saved T
	if s.Get(key, &saved) {
		v = saved
	}
	st := state.New(v)
	st.Watch(func(v T) { s.Set(key, v) })
	return st
}

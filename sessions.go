/*
 * Copyright 2023 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * The MIT License (MIT)
 *
 * Copyright (c) 2016 Bo-Yi Wu
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
* This file may have been modified by CloudWeGo authors. All CloudWeGo
* Modifications are Copyright 2022 CloudWeGo Authors.
*/

package sessions

import (
	gcontext "context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
)

const (
	DefaultKey  = "github.com/hertz-contrib/sessions"
	errorFormat = "[sessions] ERROR! %s\n"
)

type Store interface {
	sessions.Store
	Options(Options)
}

// Session Wraps thinly gorilla-session methods.
// Session stores the values and optional configuration for a session.
type Session interface {
	// ID of the session, generated by stores. It should not be used for user data.
	ID() string
	// Get returns the session value associated to the given key.
	Get(key interface{}) interface{}
	// Set sets the session value associated to the given key.
	Set(key, val interface{})
	// Delete removes the session value associated to the given key.
	Delete(key interface{})
	// Clear deletes all values in the session.
	Clear()
	// AddFlash adds a flash message to the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	AddFlash(value interface{}, vars ...string)
	// Flashes returns a slice of flash messages from the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	Flashes(vars ...string) []interface{}
	// Options sets configuration for a session.
	Options(Options)
	// Save saves all sessions used during the current request.
	Save() error
}

// Deprecated: use New instead of Sessions
func Sessions(name string, store Store) app.HandlerFunc {
	return New(name, store)
}

// Deprecated: use Many instead of SessionsMany
func SessionsMany(names []string, store Store) app.HandlerFunc {
	return Many(names, store)
}

func New(name string, store Store) app.HandlerFunc {
	return func(ctx gcontext.Context, c *app.RequestContext) {
		req, _ := adaptor.GetCompatRequest(&c.Request)
		resp := adaptor.GetCompatResponseWriter(&c.Response)
		s := &session{name, req, store, nil, false, resp}
		c.Set(DefaultKey, s)
		defer context.Clear(req)
		c.Next(ctx)
		resp.WriteHeader(c.Response.StatusCode())
	}
}

func Many(names []string, store Store) app.HandlerFunc {
	return func(ctx gcontext.Context, c *app.RequestContext) {
		s := make(map[string]Session, len(names))
		req, _ := adaptor.GetCompatRequest(&c.Request)
		resp := adaptor.GetCompatResponseWriter(&c.Response)
		for _, name := range names {
			s[name] = &session{name, req, store, nil, false, resp}
		}
		c.Set(DefaultKey, s)
		defer context.Clear(req)
		c.Next(ctx)
		resp.WriteHeader(c.Response.StatusCode())
	}
}

type session struct {
	name    string
	request *http.Request
	store   Store
	session *sessions.Session
	written bool
	writer  http.ResponseWriter
}

func (s *session) ID() string {
	return s.Session().ID
}

func (s *session) Get(key interface{}) interface{} {
	return s.Session().Values[key]
}

func (s *session) Set(key, val interface{}) {
	s.Session().Values[key] = val
	s.written = true
}

func (s *session) Delete(key interface{}) {
	delete(s.Session().Values, key)
	s.written = true
}

func (s *session) Clear() {
	for key := range s.Session().Values {
		s.Delete(key)
	}
}

func (s *session) AddFlash(value interface{}, vars ...string) {
	s.Session().AddFlash(value, vars...)
	s.written = true
}

func (s *session) Flashes(vars ...string) []interface{} {
	s.written = true
	return s.Session().Flashes(vars...)
}

func (s *session) Options(options Options) {
	s.written = true
	s.Session().Options = options.ToGorillaOptions()
}

func (s *session) Save() error {
	if s.Written() {
		e := s.Session().Save(s.request, s.writer)
		if e == nil {
			s.written = false
		}
		return e
	}
	return nil
}

func (s *session) Session() *sessions.Session {
	if s.session == nil {
		var err error
		s.session, err = s.store.Get(s.request, s.name)
		if err != nil {
			hlog.Errorf(errorFormat, err)
		}
	}
	return s.session
}

func (s *session) Written() bool {
	return s.written
}

// Default shortcut to get session
func Default(c *app.RequestContext) Session {
	return c.MustGet(DefaultKey).(Session)
}

// DefaultMany shortcut to get session with given name
func DefaultMany(c *app.RequestContext, name string) Session {
	return c.MustGet(DefaultKey).(map[string]Session)[name]
}

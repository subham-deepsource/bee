// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth_test

import (
	"testing"
	"time"

	"github.com/ethersphere/bee/pkg/auth"
)

const oneHour = 1 * time.Hour

const (
	username     = "test"
	passwordHash = "$2a$12$mZIODMvjsiS2VdK1xgI1cOTizhGVNoVz2Xn48H8ddFFLzX2B3lD3m"
)

func TestAuthorize(t *testing.T) {
	a, err := auth.New(username, passwordHash, oneHour)
	if err != nil {
		t.Error(err)
	}

	tt := []struct {
		desc       string
		user, pass string
		expected   bool
	}{
		{
			desc:     "correct credentials",
			user:     "test",
			pass:     "test",
			expected: true,
		}, {
			desc:     "wrong name",
			user:     "bad",
			pass:     "test",
			expected: false,
		}, {
			desc:     "wrong password",
			user:     "test",
			pass:     "bad",
			expected: false,
		},
	}
	for _, tC := range tt {
		t.Run(tC.desc, func(t *testing.T) {
			res := a.Authorize(tC.user, tC.pass)
			if res != tC.expected {
				t.Error("unexpected result", res)
			}
		})
	}
}

func TestEnforceWithNonExistentApiKey(t *testing.T) {
	a, err := auth.New(username, passwordHash, oneHour)
	if err != nil {
		t.Error(err)
	}

	result, err := a.Enforce("non-existent", "/resource", "GET")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if result {
		t.Errorf("expected %v, got %v", false, result)
	}
}

func TestExpiry(t *testing.T) {
	oneMili := 1 * time.Millisecond

	a, err := auth.New(username, passwordHash, oneMili)
	if err != nil {
		t.Error(err)
	}

	key, err := a.AddKey("role0")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	time.Sleep(oneMili)

	result, err := a.Enforce(key, "/bytes/1", "GET")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if result {
		t.Errorf("expected %v, got %v", false, result)
	}
}

func TestEnforce(t *testing.T) {
	a, err := auth.New(username, passwordHash, oneHour)
	if err != nil {
		t.Error(err)
	}

	tt := []struct {
		desc                   string
		role, resource, action string
		expected               bool
	}{
		{
			desc:     "success",
			role:     "role2",
			resource: "/pingpong/someone",
			action:   "POST",
			expected: true,
		},
		{
			desc:     "bad role",
			role:     "role0",
			resource: "/pingpong/some-other-peer",
			action:   "POST",
		},
		{
			desc:     "bad resource",
			role:     "role2",
			resource: "/i-dont-exist",
			action:   "POST",
		},
		{
			desc:     "bad action",
			role:     "role2",
			resource: "/pingpong/someone",
			action:   "DELETE",
		},
	}

	for _, tC := range tt {
		t.Run(tC.desc, func(t *testing.T) {
			apiKey, err := a.AddKey(tC.role)

			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			result, err := a.Enforce(apiKey, tC.resource, tC.action)

			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if result != tC.expected {
				t.Errorf("request from user with %s on object %s: expected %v, got %v", tC.role, tC.resource, tC.expected, result)
			}
		})
	}
}
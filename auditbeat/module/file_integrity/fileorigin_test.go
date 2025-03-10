// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build darwin
// +build darwin

package file_integrity

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	key   = "com.apple.metadata:kMDItemWhereFroms"
	value = `62 70 6C 69 73 74 30 30 A2 01 02 5F 10 5B 68 74
			 74 70 73 3A 2F 2F 61 72 74 69 66 61 63 74 73 2E
			 65 6C 61 73 74 69 63 2E 63 6F 2F 64 6F 77 6E 6C
			 6F 61 64 73 2F 62 65 61 74 73 2F 61 75 64 69 74
			 62 65 61 74 2F 61 75 64 69 74 62 65 61 74 2D 36
			 2E 31 2E 31 2D 64 61 72 77 69 6E 2D 78 38 36 5F
			 36 34 2E 74 61 72 2E 67 7A 5F 10 30 68 74 74 70
			 73 3A 2F 2F 77 77 77 2E 65 6C 61 73 74 69 63 2E
			 63 6F 2F 64 6F 77 6E 6C 6F 61 64 73 2F 62 65 61
			 74 73 2F 61 75 64 69 74 62 65 61 74 08 0B 69 00
			 00 00 00 00 00 01 01 00 00 00 00 00 00 00 03 00
			 00 00 00 00 00 00 00 00 00 00 00 00 00 00 9C`
)

func TestDarwinWhereFroms(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Unsupported platform")
	}

	t.Run("no origin", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "no_origin")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		origin, err := GetFileOrigin(f.Name())
		require.NoError(t, err)

		assert.Len(t, origin, 0)
	})
	t.Run("valid origin", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "valid_origin")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		//nolint:gosec // Using file.Name() as an argument is safe.
		err = exec.Command("xattr", "-w", "-x", key, value, f.Name()).Run()
		require.NoError(t, err)

		origin, err := GetFileOrigin(f.Name())
		require.NoError(t, err)

		assert.Len(t, origin, 2)
		assert.Equal(t, "https://artifacts.elastic.co/downloads/beats/auditbeat/auditbeat-6.1.1-darwin-x86_64.tar.gz", origin[0])
		assert.Equal(t, "https://www.elastic.co/downloads/beats/auditbeat", origin[1])
	})
	t.Run("empty origin", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "empty_origin")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		//nolint:gosec // Using file.Name() as an argument is safe.
		err = exec.Command("xattr", "-w", "-x", key, "", f.Name()).Run()
		require.NoError(t, err)

		origin, err := GetFileOrigin(f.Name())
		require.NoError(t, err)

		assert.Len(t, origin, 0)
	})
	t.Run("bad origin", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad_origin")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		//nolint:gosec // Using file.Name() as an argument is safe.
		err = exec.Command("xattr", "-w", "-x", key, "01 23 45 67", f.Name()).Run()
		require.NoError(t, err)

		_, err = GetFileOrigin(f.Name())
		assert.Error(t, err)
	})
}

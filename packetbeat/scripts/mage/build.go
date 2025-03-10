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

package mage

import (
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild(
		// Run all builds serially to try to address failures that might be caused
		// by concurrent builds. See https://github.com/elastic/beats/issues/24304.
		devtools.Serially(),

		devtools.ImageSelector(func(platform string) (string, error) {
			image, err := devtools.CrossBuildImage(platform)
			if err != nil {
				return "", err
			}
			if platform == "linux/386" {
				// Use Debian 9 because the linux/386 build needs an older glibc
				// to remain compatible with CentOS 7 (glibc 2.17).
				image = strings.ReplaceAll(image, "main-debian10", "main-debian9")
			}
			return image, nil
		}),
	)
}

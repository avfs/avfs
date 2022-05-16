//
//  Copyright 2022 The AVFS authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//  	http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package avfs_test

import (
	"testing"

	"github.com/avfs/avfs"
)

func TestPathIterator(t *testing.T) {
	piTests := []struct {
		path  string
		parts []string
	}{
		{
			path:  "/",
			parts: nil,
		},
		{
			path:  "/Hello",
			parts: []string{"Hello"},
		},
		{
			path:  "/Hello/World",
			parts: []string{"Hello", "World"},
		},
		{
			path:  "/एक/दो/तीन/चार/पांच",
			parts: []string{"एक", "दो", "तीन", "चार", "पांच"},
		},
	}

	ut := avfs.OSUtils

	for _, piTest := range piTests {
		path := ut.FromUnixPath("C:", piTest.path)
		i := 0

		pi := ut.NewPathIterator(path)
		for ; pi.Next(); i++ {
			if i >= len(piTest.parts) {
				continue
			}

			if pi.Part() != piTest.parts[i] {
				t.Errorf("%s : want Part %d to be %s, got %s", path, i, piTest.parts[i], pi.Part())
			}

			if pi.Left() != piTest.path[:pi.Start()] {
				t.Errorf("%s : want Left %d to be %s, got %s", path, i, piTest.path[:pi.Start()], pi.Left())
			}

			if pi.LeftPart() != piTest.path[:pi.End()] {
				t.Errorf("%s : want LeftPart %d to be %s, got %s", path, i, piTest.path[:pi.End()], pi.LeftPart())
			}

			if pi.Right() != piTest.path[pi.End():] {
				t.Errorf("%s : want Right %d to be %s, got %s", path, i, piTest.path[pi.End():], pi.Right())
			}

			if pi.RightPart() != piTest.path[pi.Start():] {
				t.Errorf("%s : want Right %d to be %s, got %s", path, i, piTest.path[pi.Start():], pi.RightPart())
			}
		}

		if i != len(piTest.parts) {
			t.Errorf("%s : want %d parts, got %d parts", path, len(piTest.parts), i)
		}
	}
}

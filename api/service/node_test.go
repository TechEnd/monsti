// Copyright 2012-2013 Christian Neumann
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"testing"
)

func TestNodeName(t *testing.T) {
	tests := []struct{ path, name string }{
		{"", ""},
		{"/", ""},
		{"/foo", "foo"},
		{"/foo/", "foo"},
		{"/foo/bar", "bar"},
		{"/foo/bar/", "bar"},
	}
	for _, test := range tests {
		node := Node{Path: test.path}
		name := node.Name()
		if name != test.name {
			t.Errorf(`%v.Name() = %q, should be %q`, node, name, test.name)
		}
	}
}

func TestFields(t *testing.T) {
	fields := []Field{
		new(TextField),
		new(HTMLField),
		new(FileField),
		new(DateTimeField),
	}
	for _, field := range fields {
		out := field.Dump()
		field.Load(out)
		out2 := field.Dump()
		if out != out2 {
			t.Errorf("Dump/Load/Dump: %q != %q", out, out2)
		}
	}
}

// This file is part of Monsti, a web content management system.
// Copyright 2012-2013 Christian Neumann
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"fmt"

	"github.com/chrneumann/mimemail"
)

// Send given Monsti.
func (s *MonstiClient) SendMail(m *mimemail.Mail) error {
	if s.Error != nil {
		return s.Error
	}
	var reply int
	if err := s.RPCClient.Call("Monsti.SendMail", m, &reply); err != nil {
		return fmt.Errorf("service: Monsti.SendMail error: %v", err)
	}
	return nil
}

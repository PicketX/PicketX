// SPDX-License-Identifier: Apache-2.0
package management

import "github.com/PicketX/PicketX/internal/account"

type accountDirectory interface {
	Accounts() (account.Page, error)
	SaveAccount(uint64, string, string, account.Change) (account.Account, uint64, error)
}

func (s *Service) Accounts(p Principal) (account.Page, error) {
	if err := s.Check(p, "account:read"); err != nil {
		return account.Page{}, err
	}
	directory, ok := s.access.(accountDirectory)
	if !ok {
		return account.Page{}, ErrUnavailable
	}
	return directory.Accounts()
}

func (s *Service) SaveAccount(p Principal, expected uint64, id string, change account.Change) (account.Account, uint64, error) {
	if err := s.Check(p, "account:write"); err != nil {
		return account.Account{}, 0, err
	}
	directory, ok := s.access.(accountDirectory)
	if !ok {
		return account.Account{}, 0, ErrUnavailable
	}
	return directory.SaveAccount(expected, p.ID, id, change)
}

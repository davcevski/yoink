//go:build darwin

package keyring

import (
	keychain "github.com/keybase/go-keychain"
)

const (
	service = "com.yoink"
	account = "encryption-key"
	label   = "yoink clipboard encryption key"
)

// Keychain is the macOS login-Keychain backed Keyring used in production.
type Keychain struct{}

// New returns the default macOS Keychain provider.
func New() Keychain { return Keychain{} }

func baseQuery() keychain.Item {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	return item
}

// Get reads the encryption key, returning ErrNotFound when it is absent.
func (Keychain) Get() ([]byte, error) {
	q := baseQuery()
	q.SetMatchLimit(keychain.MatchLimitOne)
	q.SetReturnData(true)
	results, err := keychain.QueryItem(q)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results[0].Data, nil
}

// Set stores key, creating the item or updating it in place if it already
// exists. The item is accessible only when the Keychain is unlocked and is not
// synchronized to iCloud.
func (Keychain) Set(key []byte) error {
	item := baseQuery()
	item.SetLabel(label)
	item.SetData(key)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	item.SetSynchronizable(keychain.SynchronizableNo)

	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		update := keychain.NewItem()
		update.SetData(key)
		return keychain.UpdateItem(baseQuery(), update)
	}
	return err
}

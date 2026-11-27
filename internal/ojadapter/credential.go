package ojadapter

import "encoding/json"

const credentialService = "com.routilet.companion.accoding"
const credentialAccount = "default"
const credentialLimit = 2400

func encodeCredential(c credentials) ([]byte, error) {
	b, e := json.Marshal(c)
	if e != nil || len(b) > credentialLimit || c.Username == "" || c.Password == "" {
		return nil, errCredentialStore
	}
	return b, nil
}
func decodeCredential(b []byte) (credentials, error) {
	var c credentials
	if len(b) > credentialLimit || json.Unmarshal(b, &c) != nil || c.Username == "" || c.Password == "" {
		return credentials{}, errCredentialStore
	}
	return c, nil
}

type unavailableStore struct{}

func (unavailableStore) Supported() bool            { return false }
func (unavailableStore) Load() (credentials, error) { return credentials{}, errCredentialStore }
func (unavailableStore) Save(credentials) error     { return errCredentialStore }
func (unavailableStore) Delete() error              { return errCredentialStore }

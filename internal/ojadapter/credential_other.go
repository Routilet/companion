//go:build !windows && (!darwin || !cgo)

package ojadapter

func systemCredentialStore() credentialStore { return unavailableStore{} }

func systemCredentialStoreFor(string) credentialStore { return unavailableStore{} }

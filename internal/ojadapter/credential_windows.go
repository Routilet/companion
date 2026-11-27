//go:build windows

package ojadapter

import (
	"runtime"
	"syscall"
	"unsafe"
)

// Windows Credential Manager, generic credential scoped to current user.
// No subprocess, environment, argv, or plaintext file is used.
var advapi = syscall.NewLazyDLL("Advapi32.dll")
var credRead = advapi.NewProc("CredReadW")
var credWrite = advapi.NewProc("CredWriteW")
var credDelete = advapi.NewProc("CredDeleteW")
var credFree = advapi.NewProc("CredFree")

type winCredential struct {
	Flags, Type             uint32
	TargetName, Comment     *uint16
	LastWritten             syscall.Filetime
	BlobSize                uint32
	Blob                    *byte
	Persist, AttributeCount uint32
	Attributes              uintptr
	TargetAlias, UserName   *uint16
}
type winStore struct{ service string }

func systemCredentialStore() credentialStore                  { return systemCredentialStoreFor(credentialService) }
func systemCredentialStoreFor(service string) credentialStore { return winStore{service} }
func (winStore) Supported() bool                              { return true }
func targetName(service string) *uint16                       { p, _ := syscall.UTF16PtrFromString(service); return p }
func (s winStore) Load() (credentials, error) {
	var c *winCredential
	r, _, e := credRead.Call(uintptr(unsafe.Pointer(targetName(s.service))), 1, 0, uintptr(unsafe.Pointer(&c)))
	if r == 0 {
		if e == syscall.Errno(1168) {
			return credentials{}, errNoCredential
		}
		return credentials{}, errCredentialStore
	}
	defer credFree.Call(uintptr(unsafe.Pointer(c)))
	if c == nil || c.Blob == nil || c.BlobSize == 0 || c.BlobSize > credentialLimit {
		return credentials{}, errCredentialStore
	}
	return decodeCredential(unsafe.Slice(c.Blob, int(c.BlobSize)))
}
func (s winStore) Save(v credentials) error {
	b, e := encodeCredential(v)
	if e != nil {
		return e
	}
	defer clear(b)
	c := winCredential{Type: 1, TargetName: targetName(s.service), BlobSize: uint32(len(b)), Blob: &b[0], Persist: 2}
	r, _, _ := credWrite.Call(uintptr(unsafe.Pointer(&c)), 0)
	runtime.KeepAlive(c)
	runtime.KeepAlive(b)
	if r == 0 {
		return errCredentialStore
	}
	return nil
}
func (s winStore) Delete() error {
	r, _, e := credDelete.Call(uintptr(unsafe.Pointer(targetName(s.service))), 1, 0)
	if r == 0 && e != syscall.Errno(1168) {
		return errCredentialStore
	}
	return nil
}

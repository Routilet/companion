//go:build darwin && cgo

package ojadapter

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>
static OSStatus read_secret(const char *service, const char *account, UInt32 *len, void **data) {
 return SecKeychainFindGenericPassword(NULL,(UInt32)strlen(service),service,(UInt32)strlen(account),account,len,data,NULL);
}
static OSStatus write_secret(const char *service,const char *account,UInt32 len,const void *data){
 SecKeychainItemRef item=NULL;
 OSStatus status=SecKeychainFindGenericPassword(NULL,(UInt32)strlen(service),service,(UInt32)strlen(account),account,NULL,NULL,&item);
 if(status==errSecSuccess){status=SecKeychainItemModifyAttributesAndData(item,NULL,len,data);CFRelease(item);return status;}
 if(status!=errSecItemNotFound)return status;
 return SecKeychainAddGenericPassword(NULL,(UInt32)strlen(service),service,(UInt32)strlen(account),account,len,data,NULL);
}
static OSStatus delete_secret(const char *service,const char *account){
 SecKeychainItemRef item=NULL;
 OSStatus status=SecKeychainFindGenericPassword(NULL,(UInt32)strlen(service),service,(UInt32)strlen(account),account,NULL,NULL,&item);
 if(status==errSecSuccess){status=SecKeychainItemDelete(item);CFRelease(item);}
 return status;
}
*/
import "C"
import "unsafe"

type macStore struct{ service string }

func systemCredentialStore() credentialStore                  { return systemCredentialStoreFor(credentialService) }
func systemCredentialStoreFor(service string) credentialStore { return macStore{service} }
func (macStore) Supported() bool                              { return true }
func keys(service string) (*C.char, *C.char, func()) {
	s, a := C.CString(service), C.CString(credentialAccount)
	return s, a, func() { C.free(unsafe.Pointer(s)); C.free(unsafe.Pointer(a)) }
}
func (m macStore) Load() (credentials, error) {
	s, a, done := keys(m.service)
	defer done()
	var n C.UInt32
	var data unsafe.Pointer
	st := C.read_secret(s, a, &n, &data)
	if st == C.errSecItemNotFound {
		return credentials{}, errNoCredential
	}
	if st != C.errSecSuccess {
		return credentials{}, errCredentialStore
	}
	defer C.SecKeychainItemFreeContent(nil, data)
	if n == 0 || n > credentialLimit || data == nil {
		return credentials{}, errCredentialStore
	}
	b := C.GoBytes(data, C.int(n))
	defer clear(b)
	return decodeCredential(b)
}
func (m macStore) Save(v credentials) error {
	b, e := encodeCredential(v)
	if e != nil {
		return e
	}
	defer clear(b)
	s, a, done := keys(m.service)
	defer done()
	if C.write_secret(s, a, C.UInt32(len(b)), unsafe.Pointer(&b[0])) != C.errSecSuccess {
		return errCredentialStore
	}
	return nil
}
func (m macStore) Delete() error {
	s, a, done := keys(m.service)
	defer done()
	st := C.delete_secret(s, a)
	if st != C.errSecSuccess && st != C.errSecItemNotFound {
		return errCredentialStore
	}
	return nil
}

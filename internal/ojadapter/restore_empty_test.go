package ojadapter
import("context";"testing")
func TestRestoreWithoutStoredCredentialIsNotFailure(t *testing.T){a:=NewAccounts(accountFixture(t));a.store=&memoryStore{};defer a.Close();v,e:=a.Restore(context.Background());if e!=nil||v.State!="signed_out"||a.session!=nil{t.Fatal(v,e)}}

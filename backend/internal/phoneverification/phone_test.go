package phoneverification

import "testing"

func TestPhoneProtection(t *testing.T) {
	phone, err := Normalize("+86 138-0013-8000")
	if err != nil || phone != "13800138000" {
		t.Fatalf("normalize failed: %q %v", phone, err)
	}
	if Mask(phone) != "138****8000" {
		t.Fatal("mask failed")
	}
	if Hash(phone, "secret-a") == Hash(phone, "secret-b") {
		t.Fatal("hash must be keyed")
	}
	encrypted, err := Encrypt(phone, "a sufficiently long encryption secret")
	if err != nil || encrypted == "" || encrypted == phone {
		t.Fatal("phone encryption failed")
	}
}

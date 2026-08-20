package crypto

import "testing"

func TestSignVerifyRoundTrip(t *testing.T) {
	priv, pub, err := GenerateSigningKeypair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	data := []byte("the exact bytes an upload covers")

	sig, err := Sign(data, priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	pk, err := ParseSigningPublicKey(pub)
	if err != nil {
		t.Fatalf("parse pub: %v", err)
	}
	if err := Verify(data, sig, pk); err != nil {
		t.Fatalf("verify should succeed: %v", err)
	}
}

func TestVerifyRejectsTamperedData(t *testing.T) {
	priv, pub, _ := GenerateSigningKeypair()
	sig, _ := Sign([]byte("original"), priv)
	pk, _ := ParseSigningPublicKey(pub)
	if err := Verify([]byte("tampered"), sig, pk); err == nil {
		t.Fatal("verify must fail when the data changed")
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	priv, _, _ := GenerateSigningKeypair()
	_, otherPub, _ := GenerateSigningKeypair()
	sig, _ := Sign([]byte("msg"), priv)
	pk, _ := ParseSigningPublicKey(otherPub)
	if err := Verify([]byte("msg"), sig, pk); err == nil {
		t.Fatal("verify must fail with a non-matching public key")
	}
}

package session

import (
	"github.com/dgrijalva/jwt-go"
	"testing"
)

func Test_NewToken(t *testing.T) {
	var (
		token string
		err   error
	)
	if token, err = NewToken("", map[string]interface{}{
		TokenTagUid: "sometext",
	}, nil); err != nil {
		t.Fatal(err.Error())
		return
	} else {
		println("token:", token)
	}
}

func Test_ParseTokenString(t *testing.T) {
	var (
		tokenStr = "eyJhbGciOiJIUzI1NiIsImtpZCI6IiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NTI0MTM0NzUsInVpZCI6InNvbWV0ZXh0In0.EQ7aU7O-6zuouae_Dq1dveAeRbrfFlb1X-_I4-VGyw4"
		token    *jwt.Token
		err      error
	)

	if token, err = ParseTokenString(tokenStr); err != nil {
		t.Fatal(err.Error())
		return
	} else {
		println("value:", token.Claims[TokenTagUid].(string))
	}
}

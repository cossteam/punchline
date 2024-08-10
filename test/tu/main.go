package main

import (
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

func main() {

}

func VerifyPassword(passwd string, hash string) error {
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Verify(hash, []byte(passwd))
}

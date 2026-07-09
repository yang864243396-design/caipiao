package member

import "golang.org/x/crypto/bcrypt"

func verifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

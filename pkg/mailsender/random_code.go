package mailsender

import (
	"crypto/rand"
	"math/big"
)

func RandomRumber() *big.Int {
	min := big.NewInt(100000)
	max := big.NewInt(900000)
	randomNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return new(big.Int).Add(randomNum, min)
}

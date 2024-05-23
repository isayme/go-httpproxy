package httpproxy

import (
	"math/rand"
)

const RAND_SEQ_ID_LEN = 12

var randLetters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var randLettersLen = len(randLetters)

func randSeqId() string {
	b := make([]rune, RAND_SEQ_ID_LEN)
	for i := range b {
		b[i] = randLetters[rand.Intn(randLettersLen)]
	}
	return string(b)
}

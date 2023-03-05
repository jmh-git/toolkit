package toolkit

import (
	"crypto/rand"
	m "math/rand"
	"time"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"
const NUM_NONALPHA = 12

// trand is a local wrapper of Rand used to generate random numbers out of package math/rand.
var trand *m.Rand

// init sets up the random generator with a unique Source for this instance of the module.
func init() {
	trand = m.New(m.NewSource(time.Now().Unix()))
}

// Tools is the type used to instantiate this module.
type Tools struct{}

// RandomStringWithAlpha returns a string of size length consisting of random characters. The string
// doesn't start with a non-alphabetic character.
func (t *Tools) RandomStringWithAlpha(length int) string {
	charPool := []rune(randomStringSource)
	result := make([]rune, length)
	for i := range result {
		num := len(charPool) // num = all chars from charPool
		if i == 0 {
			num -= NUM_NONALPHA // num = only alphabetic chars from charPool for first character
		}
		result[i] = charPool[trand.Intn(num)]
	}
	return string(result)
}

// RandomString returns a string of size length consisting of random characters.
func (t *Tools) RandomString(length int) string {
	s, r := make([]rune, length), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

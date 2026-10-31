package discovery

import (
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
)

func NewInstanceName() string {
	pet := petname.New(nil)

	return strings.ToTitle(pet.Adjective()) + strings.ToTitle(pet.Name())
}

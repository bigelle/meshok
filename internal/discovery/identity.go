package discovery

import (
	petname "github.com/dustinkirkland/golang-petname"
)

func NewInstanceName() string {
	pet := petname.New(nil)

	return pet.Adjective() + "_" + pet.Adverb()
}

package discovery

import (
	petname "github.com/dustinkirkland/golang-petname"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewInstanceName() string {
	pet := petname.New(nil)

	caser := cases.Title(language.English)
	adj := caser.String(pet.Adjective())
	name := caser.String(pet.Name())
	return adj + name
}

package api

import (
	"fmt"
	"time"
)

type Page struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	Main      string    `json:"main"`
	DateModif time.Time `json:"datemodif"`
}

func (p *Page) Primary() string {
	switch p.Version {
	case 1:
		return p.Main
	case 2:
		return p.Content
	default:
		panic(fmt.Sprintf("Unsuported version: %d", p.Version))
	}
}

func (p *Page) SetPrimary(primary string) {
	switch p.Version {
	case 1:
		p.Main = primary
	case 2:
		p.Content = primary
	default:
		panic(fmt.Sprintf("Unsuported version: %d", p.Version))
	}
}

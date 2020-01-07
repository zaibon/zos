package main

import "fmt"

type triangle struct {
	base    float64
	hauteur float64
}
type square struct {
	cote float64
}

func (t triangle) getArea() float64 {
	return (t.base * t.hauteur) / 2
}

func (s square) getArea() float64 {
	return (s.cote * s.cote)
}

func main() {
	t := triangle{base: 5, hauteur: 3}
	s := square{cote: 5}
	fmt.Printf("t area:%f carea:%f\n", t.getArea(), s.getArea())

}

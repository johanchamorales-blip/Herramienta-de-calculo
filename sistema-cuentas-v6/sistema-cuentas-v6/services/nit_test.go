package services

import "testing"

func TestValidarNIT(t *testing.T) {
	validos := []string{"3602978-5", "36029785", "1234567-9"}
	for _, nit := range validos {
		if err := ValidarNIT(nit); err != nil {
			t.Fatalf("%s debería ser válido: %v", nit, err)
		}
	}
	invalidos := []string{"3602978-4", "", "CF", "abc-1"}
	for _, nit := range invalidos {
		if err := ValidarNIT(nit); err == nil {
			t.Fatalf("%s no debería aceptarse", nit)
		}
	}
}

func TestFormatearNIT(t *testing.T) {
	if got := FormatearNIT("36029785"); got != "3602978-5" {
		t.Fatalf("formato inesperado: %s", got)
	}
}

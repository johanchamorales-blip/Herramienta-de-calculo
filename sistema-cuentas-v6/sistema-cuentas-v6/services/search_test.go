package services

import (
	"testing"

	"sistema-cuentas/models"
)

func TestBuscarToleraErroresOrtograficos(t *testing.T) {
	coops := []models.Cooperativa{{ID: 1, Nombre: "Chicoj", NIT: "1234", Direccion: "Zona 1"}}
	bancos := []models.Banco{{ID: 1, Nombre: "Banco Demo"}}
	cuentas := []models.Cuenta{{ID: 1, BancoID: 1, CooperativaID: 1, Numero: "001-123"}}
	r := Buscar("chhicoj", coops, bancos, cuentas, nil, nil)
	if len(r.Cooperativas) != 1 {
		t.Fatalf("se esperaba 1 cooperativa, se obtuvieron %d", len(r.Cooperativas))
	}
	if r.Cooperativas[0].Puntaje < 80 {
		t.Fatalf("puntaje inesperado: %d", r.Cooperativas[0].Puntaje)
	}
	r = Buscar("001123", coops, bancos, cuentas, nil, nil)
	if len(r.Cuentas) != 1 {
		t.Fatalf("se esperaba 1 cuenta, se obtuvieron %d", len(r.Cuentas))
	}
}

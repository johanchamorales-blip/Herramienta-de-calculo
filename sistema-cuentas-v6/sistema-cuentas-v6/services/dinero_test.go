package services

import "testing"

func TestQRedondeaACentavos(t *testing.T) {
	casos := map[float64]float64{
		0.1 + 0.2:  0.3,
		2.675:      2.68,
		-2.675:     -2.68,
		1234.56789: 1234.57,
	}
	for entrada, esperado := range casos {
		if got := Q(entrada); got != esperado {
			t.Fatalf("Q(%v) = %v, se esperaba %v", entrada, got, esperado)
		}
	}
}

func TestSumaNoAcumulaErrorBinario(t *testing.T) {
	total := 0.0
	for i := 0; i < 1000; i++ {
		total = Suma(total, 0.1)
	}
	if total != 100 {
		t.Fatalf("sumar 0.10 mil veces debe dar 100 exacto, dio %v", total)
	}
}

func TestIgualesToleraMedioCentavo(t *testing.T) {
	if !Iguales(10.001, 10) {
		t.Fatal("una diferencia menor a medio centavo debe considerarse igual")
	}
	if Iguales(10.01, 10) {
		t.Fatal("un centavo de diferencia no es igual")
	}
}

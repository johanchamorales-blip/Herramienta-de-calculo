package services

import "math"

// Q redondea un importe a dos decimales usando redondeo comercial
// (medio hacia arriba en valor absoluto).
//
// Los float64 acumulan error: 0.1 + 0.2 da 0.30000000000000004 y tras cientos
// de movimientos el saldo corrido se desvía de los centavos reales. Todas las
// sumas y restas de dinero del sistema pasan por aquí.
func Q(valor float64) float64 {
	if math.IsNaN(valor) || math.IsInf(valor, 0) {
		return 0
	}
	// Se trabaja en centavos para evitar el error de representación binaria.
	centavos := valor * 100
	redondeado := math.Floor(math.Abs(centavos) + 0.5)
	if centavos < 0 {
		redondeado = -redondeado
	}
	return redondeado / 100
}

// Suma acumula importes redondeando en cada paso.
func Suma(valores ...float64) float64 {
	var total float64
	for _, v := range valores {
		total = Q(total + Q(v))
	}
	return total
}

// Iguales compara importes con tolerancia de medio centavo.
func Iguales(a, b float64) bool { return math.Abs(Q(a)-Q(b)) < 0.005 }

// EsCero indica si un importe es cero con tolerancia de medio centavo.
func EsCero(valor float64) bool { return Iguales(valor, 0) }

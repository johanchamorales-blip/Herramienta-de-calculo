package services

import "sistema-cuentas/models"

// SaldoDeCuenta recalcula el saldo de una cuenta desde el saldo inicial y todos
// sus movimientos vigentes. Es la única fuente de verdad del saldo: así una
// anulación, una corrección o un archivo JSON editado a mano nunca dejan el
// saldo guardado fuera de sincronía con el libro.
func SaldoDeCuenta(cuenta models.Cuenta, movimientos []models.Movimiento) float64 {
	saldo := Q(cuenta.SaldoInicial)
	for _, m := range movimientos {
		if m.CuentaID != cuenta.ID {
			continue
		}
		saldo = saldoDespuesMovimiento(saldo, m)
	}
	return saldo
}

// RecalcularSaldos actualiza el saldo de todas las cuentas recibidas.
func RecalcularSaldos(cuentas []models.Cuenta, movimientos []models.Movimiento) {
	for i := range cuentas {
		cuentas[i].SaldoActual = SaldoDeCuenta(cuentas[i], movimientos)
	}
}

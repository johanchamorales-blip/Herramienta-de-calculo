package services

import (
	"testing"
	"time"

	"sistema-cuentas/models"
)

func TestConciliacionSeparaLibrosDeBanco(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	movimientos := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.August, 3), Monto: 500, Estado: models.EstadoIngresoActivo},
		{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.August, 10), Monto: 200, Estado: models.EstadoEgresoEmitido, NumeroDocumento: "100"},
		{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.August, 15), Monto: 50, Estado: models.EstadoEgresoCobrado, NumeroDocumento: "101", FechaCobro: models.NuevaFecha(2026, time.August, 20)},
		{ID: 4, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.September, 2), Monto: 100, Estado: models.EstadoIngresoActivo},
	}

	// Libros al 31/08: 1000 + 500 - 200 - 50 = 1250.
	// Banco: 1450 (estado de cuenta) - 200 (cheque en circulación) = 1250.
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, 1450, 0, 0, 0, 0, "Guatemala, 31/08/2026", "", "ana", time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.SaldoLibros != 1250 {
		t.Fatalf("saldo en libros = %v", r.Reporte.SaldoLibros)
	}
	if r.Reporte.ChequesCirculacion != 200 {
		t.Fatalf("cheques en circulación = %v", r.Reporte.ChequesCirculacion)
	}
	if r.Reporte.SaldoConciliado != 1250 || !r.Cuadrada {
		t.Fatalf("la conciliación debía cuadrar: %+v", r.Reporte)
	}
	if len(r.Cheques) != 1 || r.Cheques[0].NumeroDocumento != "100" {
		t.Fatalf("cheques en circulación incorrectos: %+v", r.Cheques)
	}
}

func TestConciliacionAplicaNotasDelLadoDeLosLibros(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	// Nota de débito de 25 (comisión que el banco ya cobró y los libros no tienen).
	r, err := GenerarConciliacion(cuenta, nil, 8, 2026, 975, 0, 25, 0, 0, "", "", "ana", time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.SaldoLibrosAjustado != 975 {
		t.Fatalf("libros ajustados = %v", r.Reporte.SaldoLibrosAjustado)
	}
	if !r.Cuadrada {
		t.Fatalf("debía cuadrar: diferencia %v", r.Reporte.DiferenciaConLibros)
	}
}

func TestConciliacionCuentaChequeCobradoDespuesDelCierre(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 500}
	movimientos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, NumeroDocumento: "9",
		FechaOperacion: models.NuevaFecha(2026, time.August, 28),
		Estado:         models.EstadoEgresoCobrado,
		FechaCobro:     models.NuevaFecha(2026, time.September, 3),
	}}
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, 500, 0, 0, 0, 0, "", "", "ana", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.ChequesCirculacion != 100 {
		t.Fatalf("el cheque cobrado en septiembre seguía en circulación al 31/08: %v", r.Reporte.ChequesCirculacion)
	}
	if !r.Cuadrada {
		t.Fatalf("debía cuadrar: %+v", r.Reporte)
	}
}

func TestConciliacionExcluyeChequeAnuladoOCobradoAntesDelCierre(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	movimientos := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 75, NumeroDocumento: "200", FechaOperacion: models.NuevaFecha(2026, time.August, 5), Estado: models.EstadoEgresoAnulado, MotivoAnulacion: "Cheque cancelado", FechaAnulacion: time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)},
		{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 125, NumeroDocumento: "201", FechaOperacion: models.NuevaFecha(2026, time.August, 7), Estado: models.EstadoEgresoCobrado, FechaCobro: models.NuevaFecha(2026, time.August, 20)},
		{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 150, NumeroDocumento: "202", FechaOperacion: models.NuevaFecha(2026, time.August, 25), Estado: models.EstadoEgresoEmitido},
	}
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, 1000, 0, 0, 0, 0, "", "", "ana", time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.ChequesCirculacion != 150 {
		t.Fatalf("solo el cheque emitido debía quedar en circulación: %v", r.Reporte.ChequesCirculacion)
	}
	if len(r.Cheques) != 1 || r.Cheques[0].NumeroDocumento != "202" {
		t.Fatalf("cheques en circulación incorrectos: %+v", r.Cheques)
	}
}

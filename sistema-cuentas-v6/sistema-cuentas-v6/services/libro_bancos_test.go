package services

import (
	"testing"
	"time"

	"sistema-cuentas/models"
)

func TestGenerarLibroBancosSinFiltro(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, Numero: "001", SaldoInicial: 1000}
	cooperativa := models.Cooperativa{ID: 1, Nombre: "Coop", Direccion: "Zona 1", NIT: "36029785"}
	banco := models.Banco{ID: 1, Nombre: "Banco"}
	m1 := models.Movimiento{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.January, 10), NumeroDocumento: "10", Concepto: "Depósito", Monto: 500, Estado: models.EstadoIngresoActivo}
	m2 := models.Movimiento{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.January, 12), NumeroDocumento: "11", Concepto: "Pago", Monto: 200, Estado: models.EstadoEgresoEmitido}

	reporte := GenerarLibroBancos(cuenta, cooperativa, banco, []models.Movimiento{m2, m1}, 0, 0, 1, 50)
	if reporte.Meta.TotalRegistros != 2 {
		t.Fatalf("esperaba 2 registros, obtuve %d", reporte.Meta.TotalRegistros)
	}
	if reporte.Meta.TotalDepositos != 500 || reporte.Meta.TotalCheques != 200 {
		t.Fatalf("totales incorrectos: depósitos=%v cheques=%v", reporte.Meta.TotalDepositos, reporte.Meta.TotalCheques)
	}
	if reporte.Meta.SaldoFinalPeriodo != 1300 {
		t.Fatalf("saldo final incorrecto: %v", reporte.Meta.SaldoFinalPeriodo)
	}
	if reporte.Meta.NIT != "3602978-5" {
		t.Fatalf("el encabezado debe llevar el NIT formateado: %s", reporte.Meta.NIT)
	}
	if reporte.Filas[0].SaldoInicial != 1000 || reporte.Filas[0].SaldoActual != 1500 {
		t.Fatalf("saldo de la primera fila incorrecto: %+v", reporte.Filas[0])
	}
	if reporte.Meta.ChequesEnCirculacion != 200 {
		t.Fatalf("el cheque emitido sigue en circulación: %v", reporte.Meta.ChequesEnCirculacion)
	}
}

func TestGenerarLibroBancosPeriodoMensual(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, Numero: "001", SaldoInicial: 1000}
	movs := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.January, 31), Concepto: "Enero", Monto: 200, Estado: models.EstadoIngresoActivo},
		{ID: 2, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.February, 10), Concepto: "Febrero", Monto: 300, Estado: models.EstadoIngresoActivo},
		{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.February, 15), Concepto: "Gasto", Monto: 100, Estado: models.EstadoEgresoEmitido},
		{ID: 4, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.March, 1), Concepto: "Marzo", Monto: 1000, Estado: models.EstadoIngresoActivo},
	}

	reporte := GenerarLibroBancos(cuenta, models.Cooperativa{}, models.Banco{}, movs, 2, 2026, 1, 50)
	if reporte.Meta.SaldoInicialPeriodo != 1200 {
		t.Fatalf("saldo inicial de febrero incorrecto: %v", reporte.Meta.SaldoInicialPeriodo)
	}
	if reporte.Meta.SaldoFinalPeriodo != 1400 {
		t.Fatalf("saldo final de febrero incorrecto: %v", reporte.Meta.SaldoFinalPeriodo)
	}
	if reporte.Meta.TotalRegistros != 2 {
		t.Fatalf("esperaba 2 movimientos de febrero, obtuvo %d", reporte.Meta.TotalRegistros)
	}
}

func TestGenerarLibroBancosIgnoraAnulados(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, Numero: "001", SaldoInicial: 1000}
	movs := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.February, 10), Concepto: "Anulado", Monto: 500, Estado: models.EstadoIngresoAnulado},
		{ID: 2, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.February, 11), Concepto: "Válido", Monto: 100, Estado: models.EstadoIngresoActivo},
	}

	reporte := GenerarLibroBancos(cuenta, models.Cooperativa{}, models.Banco{}, movs, 2, 2026, 1, 50)
	if reporte.Meta.TotalRegistros != 1 || reporte.Meta.TotalDepositos != 100 {
		t.Fatalf("los anulados no deben formar parte del libro: %+v", reporte.Meta)
	}
}

// El movimiento de fin de mes no debe migrar al mes siguiente por la zona horaria.
func TestFechaDeOperacionNoSeDesplazaPorZonaHoraria(t *testing.T) {
	guatemala := time.FixedZone("GT", -6*3600)
	fecha := models.FechaDesde(time.Date(2026, 9, 30, 20, 0, 0, 0, guatemala))
	if !fecha.EnPeriodo(9, 2026) {
		t.Fatalf("el 30/09 a las 20:00 debe seguir siendo septiembre: %s", fecha)
	}
}

func TestSaldoCorridoConCentavos(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 0}
	movs := make([]models.Movimiento, 0, 10)
	for i := 0; i < 10; i++ {
		movs = append(movs, models.Movimiento{
			ID: i + 1, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 0.1,
			Estado: models.EstadoIngresoActivo, FechaOperacion: models.NuevaFecha(2026, time.March, i+1),
		})
	}
	reporte := GenerarLibroBancos(cuenta, models.Cooperativa{}, models.Banco{}, movs, 0, 0, 1, 50)
	if reporte.Meta.SaldoFinalPeriodo != 1 {
		t.Fatalf("diez depósitos de Q0.10 deben dar Q1.00 exacto: %v", reporte.Meta.SaldoFinalPeriodo)
	}
}

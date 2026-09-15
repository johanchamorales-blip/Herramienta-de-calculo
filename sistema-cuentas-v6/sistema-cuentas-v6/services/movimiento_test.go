package services

import (
	"strings"
	"testing"
	"time"

	"sistema-cuentas/models"
)

var ahora = time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)

func cuentaBase() []models.Cuenta {
	return []models.Cuenta{{ID: 1, Numero: "001", Nombre: "Cuenta operativa", SaldoInicial: 1000}}
}

func TestCrearIngresoCalculaSaldoDesdeLosMovimientos(t *testing.T) {
	resultado, err := CrearMovimiento(models.Movimiento{
		CuentaID:  1,
		Tipo:      models.TipoIngreso,
		Monto:     250.555,
		Concepto:  "Depósito",
		Remitente: "Planilla",
	}, cuentaBase(), nil, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Movimiento.Monto != 250.56 {
		t.Fatalf("el monto debió redondearse a centavos: %v", resultado.Movimiento.Monto)
	}
	if resultado.Movimiento.SaldoInicial != 1000 || resultado.Movimiento.SaldoActual != 1250.56 {
		t.Fatalf("saldo inesperado: inicial=%v actual=%v", resultado.Movimiento.SaldoInicial, resultado.Movimiento.SaldoActual)
	}
}

func TestCrearEgresoExigeBeneficiarioYDocumento(t *testing.T) {
	_, err := CrearMovimiento(models.Movimiento{
		CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, Concepto: "Pago",
	}, cuentaBase(), nil, ahora)
	if err == nil || !strings.Contains(err.Error(), "beneficiario") {
		t.Fatalf("se esperaba error de beneficiario, obtenido: %v", err)
	}
}

func TestCrearMovimientoRechazaFechaFutura(t *testing.T) {
	_, err := CrearMovimiento(models.Movimiento{
		CuentaID: 1, Tipo: models.TipoIngreso, Monto: 100, Concepto: "Depósito", Remitente: "Socio",
		FechaOperacion: models.NuevaFecha(2026, time.September, 20),
	}, cuentaBase(), nil, ahora)
	if err == nil || !strings.Contains(err.Error(), "futura") {
		t.Fatalf("se esperaba error de fecha futura, obtenido: %v", err)
	}
}

func TestCrearEgresoAvisaSaltoDeCorrelativoConDetalle(t *testing.T) {
	previos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, NumeroDocumento: "105",
		Estado: models.EstadoEgresoEmitido, Monto: 10, FechaOperacion: models.NuevaFecha(2026, time.September, 1),
	}}
	resultado, err := CrearMovimiento(models.Movimiento{
		CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, Concepto: "Pago",
		Beneficiario: "Proveedor", NumeroDocumento: "108",
		FechaOperacion: models.NuevaFecha(2026, time.September, 5),
	}, cuentaBase(), previos, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultado.Avisos) == 0 || resultado.Avisos[0].Codigo != "SALTO_CORRELATIVO" {
		t.Fatalf("se esperaba aviso de correlativo: %+v", resultado.Avisos)
	}
	if !strings.Contains(resultado.Avisos[0].Mensaje, "106, 107") {
		t.Fatalf("el aviso debe enumerar los cheques faltantes: %s", resultado.Avisos[0].Mensaje)
	}
}

func TestCrearEgresoPermiteSobregiroConAviso(t *testing.T) {
	resultado, err := CrearMovimiento(models.Movimiento{
		CuentaID: 1, Tipo: models.TipoEgreso, Monto: 1500, Concepto: "Pago mayor al saldo",
		Beneficiario: "Proveedor", NumeroDocumento: "200",
	}, cuentaBase(), nil, ahora)
	if err != nil {
		t.Fatalf("el sobregiro debe permitirse con aviso: %v", err)
	}
	if resultado.Movimiento.SaldoActual != -500 {
		t.Fatalf("saldo esperado -500, obtenido %v", resultado.Movimiento.SaldoActual)
	}
	if len(resultado.Avisos) != 1 || resultado.Avisos[0].Codigo != "SOBREGIRO" {
		t.Fatalf("se esperaba aviso de sobregiro: %+v", resultado.Avisos)
	}
}

func TestCrearMovimientoRechazaDocumentoDuplicado(t *testing.T) {
	previos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, NumeroDocumento: "500",
		Estado: models.EstadoEgresoEmitido, Monto: 10,
	}}
	_, err := CrearMovimiento(models.Movimiento{
		CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, Concepto: "Pago",
		Beneficiario: "Proveedor", NumeroDocumento: "500",
	}, cuentaBase(), previos, ahora)
	if err == nil || !strings.Contains(err.Error(), "ya está registrado") {
		t.Fatalf("se esperaba error de duplicado, obtenido: %v", err)
	}
}

func TestAnularMovimientoExigeMotivoYRecalculaSaldo(t *testing.T) {
	cuentas := cuentaBase()
	movimientos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100,
		Estado: models.EstadoEgresoEmitido, FechaOperacion: models.NuevaFecha(2026, time.September, 2),
	}}

	if _, err := AnularMovimiento(movimientos, 1, "  ", "ana", ahora); err == nil {
		t.Fatal("la anulación sin motivo debe rechazarse")
	}

	mov, err := AnularMovimiento(movimientos, 1, "cheque dañado", "ana", ahora)
	if err != nil {
		t.Fatal(err)
	}
	RecalcularSaldos(cuentas, movimientos)
	if mov.Estado != models.EstadoEgresoAnulado || cuentas[0].SaldoActual != 1000 {
		t.Fatalf("anulación incorrecta: estado=%s saldo=%v", mov.Estado, cuentas[0].SaldoActual)
	}
}

func TestNoSePuedeAnularUnChequeYaCobrado(t *testing.T) {
	movimientos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, Estado: models.EstadoEgresoCobrado,
	}}
	if _, err := AnularMovimiento(movimientos, 1, "error", "ana", ahora); err == nil {
		t.Fatal("un cheque cobrado no debe poder anularse")
	}
}

func TestMarcarChequeCobrado(t *testing.T) {
	movimientos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100,
		Estado: models.EstadoEgresoEmitido, FechaOperacion: models.NuevaFecha(2026, time.September, 2),
	}}
	if _, err := MarcarChequeCobrado(movimientos, 1, models.NuevaFecha(2026, time.September, 1), ahora); err == nil {
		t.Fatal("no debe aceptarse un cobro anterior a la emisión")
	}
	mov, err := MarcarChequeCobrado(movimientos, 1, models.NuevaFecha(2026, time.September, 8), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if mov.Estado != models.EstadoEgresoCobrado || mov.FechaCobro.String() != "2026-09-08" {
		t.Fatalf("cobro incorrecto: %+v", mov)
	}
}

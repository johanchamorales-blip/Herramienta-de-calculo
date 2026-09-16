package services

import (
	"errors"
	"fmt"
	"time"

	"sistema-cuentas/models"
)

type ResultadoConciliacion struct {
	Reporte  models.Conciliacion `json:"reporte"`
	Cheques  []models.Movimiento `json:"cheques_en_circulacion"`
	Alertas  []Aviso             `json:"alertas"`
	Cuadrada bool                `json:"cuadrada"`
}

// GenerarConciliacion compara las dos mitades de la conciliación bancaria:
//
//	Lado banco:  saldo del estado de cuenta + depósitos en tránsito - cheques en circulación
//	Lado libros: saldo en libros + notas de crédito - notas de débito ± ajustes
//
// Los cheques en circulación se determinan por el estado real del cheque y su
// fecha de cobro respecto del cierre, nunca por el simple hecho de haber sido emitidos.
func GenerarConciliacion(
	cuenta models.Cuenta,
	movimientos []models.Movimiento,
	mes int,
	anio int,
	saldoEstado float64,
	depositosTransito float64,
	notasDebito float64,
	notasCredito float64,
	ajustes float64,
	fechaLugar string,
	observaciones string,
	usuario string,
	now time.Time,
) (ResultadoConciliacion, error) {
	if mes < 1 || mes > 12 || anio < 1 {
		return ResultadoConciliacion{}, errors.New("el mes y el año son obligatorios para conciliar")
	}
	if depositosTransito < 0 || notasDebito < 0 || notasCredito < 0 {
		return ResultadoConciliacion{}, errors.New("los depósitos en tránsito y las notas se registran en positivo")
	}

	ultimoDiaPeriodo := models.NuevaFecha(anio, time.Month(mes), 1).Time.AddDate(0, 1, -1)
	cierre := models.FechaDesde(ultimoDiaPeriodo)

	saldoLibros := Q(cuenta.SaldoInicial)
	cheques := make([]models.Movimiento, 0)
	var totalCirculacion float64

	vigentes := make([]models.Movimiento, 0, len(movimientos))
	for _, m := range movimientos {
		if m.CuentaID != cuenta.ID || !m.Afecta() || m.FechaOperacion.Despues(cierre) {
			continue
		}
		vigentes = append(vigentes, m)
	}
	OrdenarCronologico(vigentes)

	for _, m := range vigentes {
		saldoLibros = saldoDespuesMovimiento(saldoLibros, m)
		if EstaEnCirculacion(m, cierre) {
			cheques = append(cheques, m)
			totalCirculacion = Suma(totalCirculacion, m.Monto)
		}
	}

	saldoBancoAjustado := Suma(saldoEstado, depositosTransito, -totalCirculacion)
	saldoLibrosAjustado := Suma(saldoLibros, notasCredito, -notasDebito, ajustes)
	diferencia := Q(saldoLibrosAjustado - saldoBancoAjustado)

	alertas := make([]Aviso, 0)
	if !EsCero(diferencia) {
		alertas = append(alertas, Aviso{
			Codigo:  "DIFERENCIA",
			Mensaje: fmt.Sprintf("los libros y el banco no cuadran por %s", FormatearQuetzales(diferencia)),
		})
	}
	if antiguos := chequesAntiguos(cheques, cierre); antiguos > 0 {
		alertas = append(alertas, Aviso{
			Codigo:  "CHEQUES_ANTIGUOS",
			Mensaje: fmt.Sprintf("%d cheque(s) llevan más de 90 días en circulación; revisa si deben caducarse", antiguos),
		})
	}

	if fechaLugar == "" {
		fechaLugar = now.Format("2006-01-02")
	}

	return ResultadoConciliacion{
		Reporte: models.Conciliacion{
			CuentaID:            cuenta.ID,
			Mes:                 mes,
			Anio:                anio,
			SaldoLibros:         saldoLibros,
			SaldoLibrosAjustado: saldoLibrosAjustado,
			SaldoEstadoCuenta:   Q(saldoEstado),
			ChequesCirculacion:  totalCirculacion,
			DepositosEnTransito: Q(depositosTransito),
			NotasDebito:         Q(notasDebito),
			NotasCredito:        Q(notasCredito),
			Ajustes:             Q(ajustes),
			SubtotalBanco:       saldoBancoAjustado,
			SaldoConciliado:     saldoBancoAjustado,
			DiferenciaConLibros: diferencia,
			FechaLugar:          fechaLugar,
			Observaciones:       observaciones,
			Usuario:             usuario,
			FechaCreacion:       now,
		},
		Cheques:  cheques,
		Alertas:  alertas,
		Cuadrada: EsCero(diferencia),
	}, nil
}

func chequesAntiguos(cheques []models.Movimiento, cierre models.Fecha) int {
	limite := cierre.Time.AddDate(0, 0, -90)
	total := 0
	for _, c := range cheques {
		if c.FechaOperacion.Time.Before(limite) {
			total++
		}
	}
	return total
}

func ValidarConciliacionGuardable(c models.Conciliacion) error {
	if c.CuentaID <= 0 {
		return errors.New("selecciona una cuenta")
	}
	if c.Mes < 1 || c.Mes > 12 || c.Anio < 1 {
		return errors.New("el mes y el año son obligatorios")
	}
	if c.DepositosEnTransito < 0 || c.NotasDebito < 0 || c.NotasCredito < 0 {
		return errors.New("los depósitos en tránsito y las notas se registran en positivo")
	}
	return nil
}

func NumeroConciliacion(items []models.Conciliacion) int {
	max := 0
	for _, item := range items {
		if item.ID > max {
			max = item.ID
		}
	}
	return max + 1
}

func ResumenConciliacion(c models.Conciliacion) string {
	return fmt.Sprintf("Cuenta %d · %04d-%02d · diferencia %s", c.CuentaID, c.Anio, c.Mes, FormatearQuetzales(c.DiferenciaConLibros))
}

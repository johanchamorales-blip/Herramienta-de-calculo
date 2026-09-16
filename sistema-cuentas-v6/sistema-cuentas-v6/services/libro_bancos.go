package services

import (
	"sort"
	"time"

	"sistema-cuentas/models"
)

type LibroBancoFila struct {
	MovimientoID    int          `json:"movimiento_id"`
	Numero          int          `json:"numero"`
	FechaRegistro   time.Time    `json:"fecha_registro"`
	FechaOperacion  models.Fecha `json:"fecha_operacion"`
	NumeroDocumento string       `json:"numero_documento"`
	Beneficiario    string       `json:"beneficiario,omitempty"`
	Remitente       string       `json:"remitente,omitempty"`
	Concepto        string       `json:"concepto"`
	Deposito        float64      `json:"deposito"`
	Cheque          float64      `json:"cheque"`
	SaldoInicial    float64      `json:"saldo_inicial"`
	SaldoActual     float64      `json:"saldo_actual"`
	Estado          string       `json:"estado"`
}

type LibroBancoMeta struct {
	CuentaID             int     `json:"cuenta_id"`
	NumeroCuenta         string  `json:"numero_cuenta"`
	NombreCuenta         string  `json:"nombre_cuenta"`
	Banco                string  `json:"banco"`
	Cooperativa          string  `json:"cooperativa"`
	Direccion            string  `json:"direccion"`
	NIT                  string  `json:"nit"`
	Mes                  int     `json:"mes"`
	Anio                 int     `json:"anio"`
	TotalDepositos       float64 `json:"total_depositos"`
	TotalCheques         float64 `json:"total_cheques"`
	ChequesEnCirculacion float64 `json:"cheques_en_circulacion"`
	SaldoInicialPeriodo  float64 `json:"saldo_inicial_periodo"`
	SaldoFinalPeriodo    float64 `json:"saldo_final_periodo"`
	TotalRegistros       int     `json:"total_registros"`
	Pagina               int     `json:"pagina"`
	TamanoPagina         int     `json:"tamano_pagina"`
	TotalPaginas         int     `json:"total_paginas"`
}

type ReporteLibroBancos struct {
	Meta  LibroBancoMeta   `json:"meta"`
	Filas []LibroBancoFila `json:"filas"`
}

// GenerarLibroBancos construye el libro de una cuenta con saldo corrido.
// El periodo es opcional: mes=0 y anio=0 devuelven todo el historial vigente.
// Si se indica solo el mes, se asume el año en curso del último movimiento.
func GenerarLibroBancos(
	cuenta models.Cuenta,
	cooperativa models.Cooperativa,
	banco models.Banco,
	movimientos []models.Movimiento,
	mes int,
	anio int,
	pagina int,
	tamanoPagina int,
) ReporteLibroBancos {
	if pagina < 1 {
		pagina = 1
	}
	if tamanoPagina <= 0 || tamanoPagina > 200 {
		tamanoPagina = 50
	}

	vigentes := make([]models.Movimiento, 0, len(movimientos))
	for _, m := range movimientos {
		if m.CuentaID != cuenta.ID || !m.Afecta() {
			continue
		}
		vigentes = append(vigentes, m)
	}
	OrdenarCronologico(vigentes)

	filtrarPeriodo := mes >= 1 && mes <= 12 && anio > 0

	saldo := Q(cuenta.SaldoInicial)
	saldoInicialPeriodo := saldo
	filas := make([]LibroBancoFila, 0)
	var totalDepositos, totalCheques, enCirculacion float64
	numero := 0

	for _, m := range vigentes {
		if filtrarPeriodo && !m.FechaOperacion.EnPeriodo(mes, anio) {
			// Los movimientos previos al periodo arrastran el saldo inicial;
			// los posteriores no se toman en cuenta.
			if m.FechaOperacion.Antes(models.NuevaFecha(anio, time.Month(mes), 1)) {
				saldo = saldoDespuesMovimiento(saldo, m)
				saldoInicialPeriodo = saldo
			}
			continue
		}

		saldoInicial := saldo
		saldo = saldoDespuesMovimiento(saldo, m)
		numero++

		fila := LibroBancoFila{
			MovimientoID:    m.ID,
			Numero:          numero,
			FechaRegistro:   m.FechaRegistro,
			FechaOperacion:  m.FechaOperacion,
			NumeroDocumento: m.NumeroDocumento,
			Beneficiario:    m.Beneficiario,
			Remitente:       m.Remitente,
			Concepto:        m.Concepto,
			SaldoInicial:    saldoInicial,
			SaldoActual:     saldo,
			Estado:          m.Estado,
		}
		switch m.Tipo {
		case models.TipoIngreso:
			fila.Deposito = m.Monto
			totalDepositos = Suma(totalDepositos, m.Monto)
		case models.TipoEgreso:
			fila.Cheque = m.Monto
			totalCheques = Suma(totalCheques, m.Monto)
			if EstaEnCirculacion(m, models.NuevaFecha(anio, time.Month(mes), 1).Time.AddDate(0, 1, -1)) {
				enCirculacion = Suma(enCirculacion, m.Monto)
			}
		}
		filas = append(filas, fila)
	}

	totalRegistros := len(filas)
	totalPaginas := 0
	if totalRegistros > 0 {
		totalPaginas = (totalRegistros + tamanoPagina - 1) / tamanoPagina
	}
	if totalPaginas > 0 && pagina > totalPaginas {
		pagina = totalPaginas
	}
	inicio := (pagina - 1) * tamanoPagina
	if inicio > totalRegistros {
		inicio = totalRegistros
	}
	fin := inicio + tamanoPagina
	if fin > totalRegistros {
		fin = totalRegistros
	}

	return ReporteLibroBancos{
		Meta: LibroBancoMeta{
			CuentaID:             cuenta.ID,
			NumeroCuenta:         cuenta.Numero,
			NombreCuenta:         cuenta.Nombre,
			Banco:                banco.Nombre,
			Cooperativa:          cooperativa.Nombre,
			Direccion:            cooperativa.Direccion,
			NIT:                  FormatearNIT(cooperativa.NIT),
			Mes:                  mes,
			Anio:                 anio,
			TotalDepositos:       totalDepositos,
			TotalCheques:         totalCheques,
			ChequesEnCirculacion: enCirculacion,
			SaldoInicialPeriodo:  saldoInicialPeriodo,
			SaldoFinalPeriodo:    saldo,
			TotalRegistros:       totalRegistros,
			Pagina:               pagina,
			TamanoPagina:         tamanoPagina,
			TotalPaginas:         totalPaginas,
		},
		Filas: filas[inicio:fin],
	}
}

// OrdenarCronologico ordena por fecha de operación y, a igual fecha, por el
// orden en que se registraron. Así el saldo corrido es estable y reproducible.
func OrdenarCronologico(movimientos []models.Movimiento) {
	sort.SliceStable(movimientos, func(i, j int) bool {
		if movimientos[i].FechaOperacion.Time.Equal(movimientos[j].FechaOperacion.Time) {
			return movimientos[i].ID < movimientos[j].ID
		}
		return movimientos[i].FechaOperacion.Antes(movimientos[j].FechaOperacion)
	})
}

func saldoDespuesMovimiento(saldo float64, m models.Movimiento) float64 {
	if !m.Afecta() {
		return Q(saldo)
	}
	switch m.Tipo {
	case models.TipoIngreso:
		return Suma(saldo, m.Monto)
	case models.TipoEgreso:
		return Suma(saldo, -m.Monto)
	}
	return Q(saldo)
}

package handlers

import (
	"net/http"
	"sort"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

type resumenCuenta struct {
	CuentaID             int     `json:"cuenta_id"`
	Numero               string  `json:"numero"`
	Nombre               string  `json:"nombre"`
	Banco                string  `json:"banco"`
	Cooperativa          string  `json:"cooperativa"`
	NIT                  string  `json:"nit"`
	Saldo                float64 `json:"saldo"`
	DepositosMes         float64 `json:"depositos_mes"`
	ChequesMes           float64 `json:"cheques_mes"`
	ChequesEnCirculacion int     `json:"cheques_en_circulacion"`
	MontoEnCirculacion   float64 `json:"monto_en_circulacion"`
	ConciliadaHasta      string  `json:"conciliada_hasta"`
}

// Resumen alimenta el panel de inicio: una foto del mes en curso por cuenta,
// más las alertas que exigen acción.
func Resumen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	ahora := time.Now().UTC()
	mes := queryInt(r, "mes", int(ahora.Month()))
	anio := queryInt(r, "anio", ahora.Year())
	if mes < 1 || mes > 12 {
		mes = int(ahora.Month())
	}
	if anio < 1 {
		anio = ahora.Year()
	}

	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	movimientos, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	conciliaciones, err := conciliacionStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	services.RecalcularSaldos(cuentas, movimientos)

	detalle := make([]resumenCuenta, 0, len(cuentas))
	var saldoTotal, depositosTotal, chequesTotal, circulacionTotal float64
	alertas := make([]services.Aviso, 0)

	for _, cuenta := range cuentas {
		cooperativa, _ := buscarCooperativa(cooperativas, cuenta.CooperativaID)
		banco, _ := buscarBanco(bancos, cuenta.BancoID)

		fila := resumenCuenta{
			CuentaID:    cuenta.ID,
			Numero:      cuenta.Numero,
			Nombre:      cuenta.Nombre,
			Banco:       banco.Nombre,
			Cooperativa: cooperativa.Nombre,
			NIT:         services.FormatearNIT(cooperativa.NIT),
			Saldo:       cuenta.SaldoActual,
		}

		for _, m := range movimientos {
			if m.CuentaID != cuenta.ID || !m.Afecta() {
				continue
			}
			if m.FechaOperacion.EnPeriodo(mes, anio) {
				switch m.Tipo {
				case models.TipoIngreso:
					fila.DepositosMes = services.Suma(fila.DepositosMes, m.Monto)
				case models.TipoEgreso:
					fila.ChequesMes = services.Suma(fila.ChequesMes, m.Monto)
				}
			}
			if m.Tipo == models.TipoEgreso && m.Estado == models.EstadoEgresoEmitido {
				fila.ChequesEnCirculacion++
				fila.MontoEnCirculacion = services.Suma(fila.MontoEnCirculacion, m.Monto)
			}
		}

		for _, c := range conciliaciones {
			if c.CuentaID != cuenta.ID {
				continue
			}
			etiqueta := periodoTexto(c.Mes, c.Anio)
			if etiqueta > fila.ConciliadaHasta {
				fila.ConciliadaHasta = etiqueta
			}
		}

		if fila.Saldo < 0 {
			alertas = append(alertas, services.Aviso{
				Codigo:  "SOBREGIRO",
				Mensaje: "La cuenta " + cuenta.Numero + " está sobregirada en " + services.FormatearQuetzales(-fila.Saldo),
			})
		}
		if fila.ConciliadaHasta == "" && fila.DepositosMes+fila.ChequesMes > 0 {
			alertas = append(alertas, services.Aviso{
				Codigo:  "SIN_CONCILIAR",
				Mensaje: "La cuenta " + cuenta.Numero + " no tiene ninguna conciliación guardada",
			})
		}

		saldoTotal = services.Suma(saldoTotal, fila.Saldo)
		depositosTotal = services.Suma(depositosTotal, fila.DepositosMes)
		chequesTotal = services.Suma(chequesTotal, fila.ChequesMes)
		circulacionTotal = services.Suma(circulacionTotal, fila.MontoEnCirculacion)
		detalle = append(detalle, fila)
	}

	sort.SliceStable(detalle, func(i, j int) bool { return detalle[i].Saldo > detalle[j].Saldo })

	vigentes := make([]models.Movimiento, 0, len(movimientos))
	for _, m := range movimientos {
		if m.Afecta() {
			vigentes = append(vigentes, m)
		}
	}
	sort.SliceStable(vigentes, func(i, j int) bool { return vigentes[i].ID > vigentes[j].ID })
	if len(vigentes) > 8 {
		vigentes = vigentes[:8]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"periodo": map[string]int{"mes": mes, "anio": anio},
		"totales": map[string]any{
			"cooperativas":       len(cooperativas),
			"bancos":             len(bancos),
			"cuentas":            len(cuentas),
			"saldo":              saldoTotal,
			"depositos_mes":      depositosTotal,
			"cheques_mes":        chequesTotal,
			"monto_en_circulacion": circulacionTotal,
		},
		"cuentas":            detalle,
		"ultimos_movimientos": vigentes,
		"alertas":            alertas,
	})
}

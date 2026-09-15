package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sistema-cuentas/services"
)

var nombresMes = []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}

func LibroBancos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta para ver el libro")
		return
	}
	mes := queryInt(r, "mes", 0)
	anio := queryInt(r, "anio", 0)
	if mes < 0 || mes > 12 || anio < 0 {
		writeError(w, http.StatusBadRequest, "el periodo indicado no es válido")
		return
	}
	if mes > 0 && anio == 0 {
		writeError(w, http.StatusBadRequest, "indica el año del periodo")
		return
	}

	cuenta, cooperativa, banco, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}

	reporte := services.GenerarLibroBancos(cuenta, cooperativa, banco, movimientos, mes, anio,
		queryInt(r, "pagina", 1), queryInt(r, "tamano", 50))
	writeJSON(w, http.StatusOK, reporte)
}

func LibroBancosCSV(w http.ResponseWriter, r *http.Request) {
	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta para exportar")
		return
	}
	cuenta, cooperativa, banco, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}

	mes := queryInt(r, "mes", 0)
	anio := queryInt(r, "anio", 0)
	// Una sola llamada con todas las filas: así los totales y el saldo corrido
	// del archivo coinciden exactamente con lo que se ve en pantalla.
	completo := services.GenerarLibroBancos(cuenta, cooperativa, banco, movimientos, mes, anio, 1, 200)
	total := completo.Meta.TotalRegistros

	filename := fmt.Sprintf("libro_bancos_%s", strings.ReplaceAll(cuenta.Numero, " ", "_"))
	if mes >= 1 && mes <= 12 && anio > 0 {
		filename += fmt.Sprintf("_%04d-%02d", anio, mes)
	}
	filename += ".csv"

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	// BOM para que Excel en Windows respete los acentos.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"LIBRO DE BANCOS"})
	_ = cw.Write([]string{"Cooperativa", completo.Meta.Cooperativa})
	_ = cw.Write([]string{"NIT", completo.Meta.NIT})
	_ = cw.Write([]string{"Dirección", completo.Meta.Direccion})
	_ = cw.Write([]string{"Cuenta", completo.Meta.NombreCuenta + " " + completo.Meta.NumeroCuenta})
	_ = cw.Write([]string{"Banco", completo.Meta.Banco})
	_ = cw.Write([]string{"Periodo", periodoTexto(mes, anio)})
	_ = cw.Write([]string{"Saldo inicial", formatearDecimal(completo.Meta.SaldoInicialPeriodo)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"No.", "Fecha", "No. de cheque o documento", "Beneficiario", "Remitente", "Concepto", "Depósitos", "Cheques", "Saldo", "Estado"})

	pagina := 1
	escritas := 0
	for escritas < total {
		parcial := services.GenerarLibroBancos(cuenta, cooperativa, banco, movimientos, mes, anio, pagina, 200)
		if len(parcial.Filas) == 0 {
			break
		}
		for _, fila := range parcial.Filas {
			_ = cw.Write([]string{
				strconv.Itoa(fila.Numero),
				fila.FechaOperacion.String(),
				fila.NumeroDocumento,
				fila.Beneficiario,
				fila.Remitente,
				fila.Concepto,
				formatearDecimal(fila.Deposito),
				formatearDecimal(fila.Cheque),
				formatearDecimal(fila.SaldoActual),
				fila.Estado,
			})
			escritas++
		}
		pagina++
	}

	_ = cw.Write(nil)
	_ = cw.Write([]string{"", "", "", "", "", "TOTALES",
		formatearDecimal(completo.Meta.TotalDepositos),
		formatearDecimal(completo.Meta.TotalCheques),
		formatearDecimal(completo.Meta.SaldoFinalPeriodo), ""})
	_ = cw.Write([]string{"Cheques en circulación", formatearDecimal(completo.Meta.ChequesEnCirculacion)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Elaboró", "Tesorero", "Vo. Bo.", "Presidente Comisión de Vigilancia"})
	cw.Flush()
}

func periodoTexto(mes, anio int) string {
	if mes < 1 || mes > 12 || anio <= 0 {
		return "Histórico completo"
	}
	return fmt.Sprintf("%s %d", nombresMes[mes], anio)
}

func formatearDecimal(v float64) string {
	return strconv.FormatFloat(services.Q(v), 'f', 2, 64)
}

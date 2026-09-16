package handlers

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func Conciliaciones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listarConciliaciones(w, r)
	case http.MethodPost:
		guardarConciliacion(w, r)
	default:
		methodNotAllowed(w)
	}
}

func listarConciliaciones(w http.ResponseWriter, r *http.Request) {
	items, err := conciliacionStore.Read(); if err != nil { responderError(w, err); return }
	if raw := strings.TrimSpace(r.URL.Query().Get("cuenta_id")); raw != "" {
		id, err := strconv.Atoi(raw); if err != nil { writeError(w, http.StatusBadRequest, "cuenta_id inválido"); return }
		filtradas := make([]models.Conciliacion, 0); for _, item := range items { if item.CuentaID == id { filtradas = append(filtradas, item) } }; items = filtradas
	}
	sort.SliceStable(items, func(i,j int) bool { if items[i].Anio == items[j].Anio { return items[i].Mes > items[j].Mes }; return items[i].Anio > items[j].Anio })
	writeJSON(w, http.StatusOK, items)
}

func obtenerConciliacion(w http.ResponseWriter, r *http.Request) {
	id := queryInt(r, "id", 0); if id <= 0 { writeError(w, http.StatusBadRequest, "indica la conciliación"); return }
	items, err := conciliacionStore.Read(); if err != nil { responderError(w, err); return }
	for _, item := range items { if item.ID == id { writeJSON(w, http.StatusOK, item); return } }
	writeError(w, http.StatusNotFound, "la conciliación no existe")
}

type entradaConciliacion struct { CuentaID int `json:"cuenta_id"`; Mes int `json:"mes"`; Anio int `json:"anio"`; SaldoEstadoCuenta float64 `json:"saldo_estado_cuenta"`; DepositosEnTransito float64 `json:"depositos_en_transito"`; NotasDebito float64 `json:"notas_debito"`; NotasCredito float64 `json:"notas_credito"`; Ajustes float64 `json:"ajustes"`; FechaLugar string `json:"fecha_lugar"`; Observaciones string `json:"observaciones"`; MovimientosBanco []models.MovimientoBanco `json:"movimientos_banco"` }

func guardarConciliacion(w http.ResponseWriter, r *http.Request) {
	var input entradaConciliacion; if err := readJSON(r, &input); err != nil { writeError(w, http.StatusBadRequest, "revisa los datos enviados"); return }
	usuario, _ := SesionDe(r)
	errTx := enTransaccion(func() error {
		cuenta, _, _, movimientos, err := contextoCuenta(input.CuentaID); if err != nil { responderError(w, err); return nil }
		resultado, err := services.GenerarConciliacion(cuenta, movimientos, input.Mes, input.Anio, input.SaldoEstadoCuenta, input.DepositosEnTransito, input.NotasDebito, input.NotasCredito, input.Ajustes, strings.TrimSpace(input.FechaLugar), strings.TrimSpace(input.Observaciones), usuario.Usuario, time.Now().UTC()); if err != nil { writeError(w, http.StatusBadRequest, err.Error()); return nil }
		if len(input.MovimientosBanco) > 0 { resultado.Reporte.MovimientosBanco = input.MovimientosBanco }
		items, err := conciliacionStore.Read(); if err != nil { return err }
		for _, existente := range items { if existente.CuentaID == input.CuentaID && existente.Mes == input.Mes && existente.Anio == input.Anio { writeError(w, http.StatusConflict, "esa cuenta ya tiene una conciliación guardada para el periodo"); return nil } }
		resultado.Reporte.ID = services.NumeroConciliacion(items); items = append(items, resultado.Reporte)
		if err := conciliacionStore.Write(items); err != nil { return err }
		auditar(r, "GUARDÓ CONCILIACIÓN", "Cuenta "+cuenta.Numero, services.ResumenConciliacion(resultado.Reporte)); writeJSON(w, http.StatusCreated, resultado); return nil
	})
	if errTx != nil { responderError(w, errTx) }
}

func ImportarEstadoCuenta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { methodNotAllowed(w); return }
	if err := r.ParseMultipartForm(20 << 20); err != nil { writeError(w, http.StatusBadRequest, "no se pudo leer el archivo"); return }
	file, header, err := r.FormFile("archivo"); if err != nil { writeError(w, http.StatusBadRequest, "selecciona un estado de cuenta"); return }; defer file.Close()
	data, err := io.ReadAll(file); if err != nil { writeError(w, http.StatusBadRequest, "no se pudo leer el archivo"); return }
	resultado, err := services.ImportarEstadoCuenta(header.Filename, data); if err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
	cuentaID, _ := strconv.Atoi(r.FormValue("cuenta_id")); if cuentaID > 0 { _,_,_,libros, e := contextoCuenta(cuentaID); if e == nil { d, nd, nc := services.CompararMovimientosBanco(resultado.MovimientosBanco, libros, resultado.Mes, resultado.Anio); resultado.DepositosEnTransito = d; resultado.NotasDebito = nd; resultado.NotasCredito = nc; for _, m := range resultado.MovimientosBanco { if m.Conciliado { resultado.MovimientosConciliados++ } }; resultado.MovimientosPendientes = resultado.MovimientosDetectados - resultado.MovimientosConciliados } }
	writeJSON(w, http.StatusOK, resultado)
}

func ConciliacionPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { methodNotAllowed(w); return }
	cuentaID := queryInt(r, "cuenta_id", 0); if cuentaID <= 0 { writeError(w, http.StatusBadRequest, "selecciona una cuenta"); return }
	mes, anio := queryInt(r,"mes",0), queryInt(r,"anio",0); valores := map[string]float64{}
	for _, clave := range []string{"saldo_estado_cuenta","depositos_en_transito","notas_debito","notas_credito","ajustes"} { valor, err := queryFloat(r, clave); if err != nil { writeError(w,http.StatusBadRequest,"el valor de "+strings.ReplaceAll(clave,"_"," ")+" no es un número"); return }; valores[clave]=valor }
	cuenta,_,_,movimientos,err := contextoCuenta(cuentaID); if err != nil { responderError(w,err); return }; usuario,_ := SesionDe(r)
	resultado,err := services.GenerarConciliacion(cuenta,movimientos,mes,anio,valores["saldo_estado_cuenta"],valores["depositos_en_transito"],valores["notas_debito"],valores["notas_credito"],valores["ajustes"],strings.TrimSpace(r.URL.Query().Get("fecha_lugar")),strings.TrimSpace(r.URL.Query().Get("observaciones")),usuario.Usuario,time.Now().UTC()); if err != nil { writeError(w,http.StatusBadRequest,err.Error()); return }; writeJSON(w,http.StatusOK,resultado)
}

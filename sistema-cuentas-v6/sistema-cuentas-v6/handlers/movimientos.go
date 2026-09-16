package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func Movimientos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listarMovimientos(w, r)
	case http.MethodPost:
		crearMovimiento(w, r)
	case http.MethodDelete:
		anularMovimiento(w, r)
	default:
		methodNotAllowed(w)
	}
}

func listarMovimientos(w http.ResponseWriter, r *http.Request) {
	items, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("cuenta_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cuenta_id inválido")
			return
		}
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.CuentaID == id {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}
	if tipo := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("tipo"))); tipo != "" {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.Tipo == tipo {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}
	if estado := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("estado"))); estado != "" {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.Estado == estado {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}

	// Los filtros de días siempre usan FechaOperacion, que es la fecha contable
	// del movimiento. FechaRegistro se conserva solo como dato de auditoría.
	desde, err := fechaConsulta(r, "desde")
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha desde no es válida")
		return
	}
	hasta, err := fechaConsulta(r, "hasta")
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha hasta no es válida")
		return
	}
	if !desde.EsVacia() && !hasta.EsVacia() && desde.Despues(hasta) {
		writeError(w, http.StatusBadRequest, "la fecha desde no puede ser posterior a la fecha hasta")
		return
	}
	if !desde.EsVacia() || !hasta.EsVacia() {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if !desde.EsVacia() && m.FechaOperacion.Antes(desde) {
				continue
			}
			if !hasta.EsVacia() && m.FechaOperacion.Despues(hasta) {
				continue
			}
			filtrados = append(filtrados, m)
		}
		items = filtrados
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].FechaOperacion.Time.Equal(items[j].FechaOperacion.Time) {
			return items[i].ID > items[j].ID
		}
		return items[i].FechaOperacion.Despues(items[j].FechaOperacion)
	})

	if limite := queryInt(r, "limite", 0); limite > 0 && len(items) > limite {
		items = items[:limite]
	}
	writeJSON(w, http.StatusOK, items)
}

func fechaConsulta(r *http.Request, nombre string) (models.Fecha, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(nombre))
	if raw == "" {
		return models.Fecha{}, nil
	}
	return models.ParsearFecha(raw)
}

func crearMovimiento(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		CuentaID           int     `json:"cuenta_id"`
		Tipo               string  `json:"tipo"`
		FechaOperacion     string  `json:"fecha_operacion"`
		FechaEmisionCheque string  `json:"fecha_emision_cheque"`
		NumeroDocumento    string  `json:"numero_documento"`
		Remitente          string  `json:"remitente"`
		Beneficiario       string  `json:"beneficiario"`
		Concepto           string  `json:"concepto"`
		Monto              float64 `json:"monto"`
	}
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}

	fechaOperacion, err := models.ParsearFecha(entrada.FechaOperacion)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de operación no es válida")
		return
	}
	fechaEmision, err := models.ParsearFecha(entrada.FechaEmisionCheque)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de emisión no es válida")
		return
	}

	usuario, _ := SesionDe(r)
	movimiento := models.Movimiento{
		CuentaID:           entrada.CuentaID,
		Tipo:               entrada.Tipo,
		FechaOperacion:     fechaOperacion,
		FechaEmisionCheque: fechaEmision,
		NumeroDocumento:    entrada.NumeroDocumento,
		Remitente:          entrada.Remitente,
		Beneficiario:       entrada.Beneficiario,
		Concepto:           entrada.Concepto,
		Monto:              entrada.Monto,
		Usuario:            usuario.Usuario,
	}

	errTx := enTransaccion(func() error {
		cuentas, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}

		resultado, err := services.CrearMovimiento(movimiento, cuentas, movimientos, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		movimientos = append(movimientos, resultado.Movimiento)
		services.RecalcularSaldos(cuentas, movimientos)

		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		if err := cuentaStore.Write(cuentas); err != nil {
			return err
		}

		cuenta, _ := buscarCuenta(cuentas, resultado.Movimiento.CuentaID)
		auditar(r, "REGISTRÓ "+resultado.Movimiento.Tipo,
			services.DocumentoMovimiento(resultado.Movimiento, cuenta.Numero),
			services.FormatearQuetzales(resultado.Movimiento.Monto)+" · "+resultado.Movimiento.Concepto)

		writeJSON(w, http.StatusCreated, map[string]any{
			"movimiento": resultado.Movimiento,
			"avisos":     resultado.Avisos,
			"saldo":      cuenta.SaldoActual,
		})
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

// CobrarCheque marca un egreso como pagado por el banco. Hasta ese momento el
// cheque se considera en circulación en la conciliación.
func CobrarCheque(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var entrada struct {
		ID         int    `json:"id"`
		FechaCobro string `json:"fecha_cobro"`
	}
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	fechaCobro, err := models.ParsearFecha(entrada.FechaCobro)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de cobro no es válida")
		return
	}

	errTx := enTransaccion(func() error {
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}
		movimiento, err := services.MarcarChequeCobrado(movimientos, entrada.ID, fechaCobro, time.Now().UTC())
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "el movimiento no existe")
				return nil
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}
		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		cuentas, _ := cuentaStore.Read()
		cuenta, _ := buscarCuenta(cuentas, movimiento.CuentaID)
		auditar(r, "REGISTRÓ COBRO DE CHEQUE",
			services.DocumentoMovimiento(movimiento, cuenta.Numero),
			"Cobrado el "+movimiento.FechaCobro.String())
		writeJSON(w, http.StatusOK, movimiento)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

func anularMovimiento(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("id")))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "indica el movimiento a anular")
		return
	}
	motivo := strings.TrimSpace(r.URL.Query().Get("motivo"))
	usuario, _ := SesionDe(r)

	errTx := enTransaccion(func() error {
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}
		movimiento, err := services.AnularMovimiento(movimientos, id, motivo, usuario.Usuario, time.Now().UTC())
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "el movimiento no existe")
				return nil
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		cuentas, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		services.RecalcularSaldos(cuentas, movimientos)

		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		if err := cuentaStore.Write(cuentas); err != nil {
			return err
		}

		cuenta, _ := buscarCuenta(cuentas, movimiento.CuentaID)
		auditar(r, "ANULÓ "+movimiento.Tipo,
			services.DocumentoMovimiento(movimiento, cuenta.Numero),
			motivo)
		writeJSON(w, http.StatusOK, movimiento)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

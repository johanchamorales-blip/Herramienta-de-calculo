package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

// Cooperativas administra el catálogo de cooperativas.
func Cooperativas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := cooperativaStore.Read()
		if err != nil {
			responderError(w, err)
			return
		}
		if q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q"))); q != "" {
			filtradas := make([]models.Cooperativa, 0)
			for _, c := range items {
				texto := strings.ToLower(c.Nombre + " " + c.NIT + " " + c.Direccion)
				if strings.Contains(texto, q) {
					filtradas = append(filtradas, c)
				}
			}
			items = filtradas
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPut:
		guardarCooperativa(w, r)
	case http.MethodDelete:
		eliminarCooperativa(w, r)
	default:
		methodNotAllowed(w)
	}
}

func guardarCooperativa(w http.ResponseWriter, r *http.Request) {
	var item models.Cooperativa
	if err := readJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	item.Nombre = strings.TrimSpace(item.Nombre)
	item.Direccion = strings.TrimSpace(item.Direccion)
	item.Telefono = strings.TrimSpace(item.Telefono)

	// 1. Guardamos el NIT con su formato correcto (con el guion)
	item.NIT = services.FormatearNIT(item.NIT)

	// 2. Usamos una variable normalizada solo para comparar duplicados
	nitNormalizado := services.NormalizarNIT(item.NIT)

	if item.Nombre == "" {
		writeError(w, http.StatusBadRequest, "el nombre de la cooperativa es obligatorio")
		return
	}
	if err := services.ValidarNIT(item.NIT); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := enTransaccion(func() error {
		items, err := cooperativaStore.Read()
		if err != nil {
			return err
		}

		// 3. Comparamos usando la versión normalizada para evitar falsos positivos
		for _, c := range items {
			if c.ID != item.ID && services.NormalizarNIT(c.NIT) == nitNormalizado {
				writeError(w, http.StatusConflict, "ya hay una cooperativa registrada con el NIT "+item.NIT)
				return nil
			}
		}

		if item.ID > 0 {
			for i := range items {
				if items[i].ID == item.ID {
					items[i] = item
					if err := cooperativaStore.Write(items); err != nil {
						return err
					}
					auditar(r, "ACTUALIZÓ COOPERATIVA", item.Nombre, "NIT "+item.NIT)
					writeJSON(w, http.StatusOK, item)
					return nil
				}
			}
			writeError(w, http.StatusNotFound, "la cooperativa no existe")
			return nil
		}

		item.ID = siguienteID(len(items), func(i int) int { return items[i].ID })
		items = append(items, item)
		if err := cooperativaStore.Write(items); err != nil {
			return err
		}
		auditar(r, "CREÓ COOPERATIVA", item.Nombre, "NIT "+item.NIT)
		writeJSON(w, http.StatusCreated, item)
		return nil
	})
	if err != nil {
		responderError(w, err)
	}
}

func eliminarCooperativa(w http.ResponseWriter, r *http.Request) {
	id := queryInt(r, "id", 0)
	if id <= 0 {
		writeError(w, http.StatusBadRequest, "indica la cooperativa a eliminar")
		return
	}
	err := enTransaccion(func() error {
		cuentas, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		for _, c := range cuentas {
			if c.CooperativaID == id {
				writeError(w, http.StatusConflict, "la cooperativa tiene cuentas bancarias asociadas; elimínalas primero")
				return nil
			}
		}
		items, err := cooperativaStore.Read()
		if err != nil {
			return err
		}
		for i := range items {
			if items[i].ID == id {
				eliminada := items[i]
				items = append(items[:i], items[i+1:]...)
				if err := cooperativaStore.Write(items); err != nil {
					return err
				}
				auditar(r, "ELIMINÓ COOPERATIVA", eliminada.Nombre, "NIT "+services.FormatearNIT(eliminada.NIT))
				writeJSON(w, http.StatusOK, eliminada)
				return nil
			}
		}
		writeError(w, http.StatusNotFound, "la cooperativa no existe")
		return nil
	})
	if err != nil {
		responderError(w, err)
	}
}

// Bancos administra el catálogo de bancos.
func Bancos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := bancoStore.Read()
		if err != nil {
			responderError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPut:
		var item models.Banco
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, "revisa los datos enviados")
			return
		}
		item.Nombre = strings.TrimSpace(item.Nombre)
		if item.Nombre == "" {
			writeError(w, http.StatusBadRequest, "el nombre del banco es obligatorio")
			return
		}
		err := enTransaccion(func() error {
			items, err := bancoStore.Read()
			if err != nil {
				return err
			}
			for _, b := range items {
				if b.ID != item.ID && strings.EqualFold(b.Nombre, item.Nombre) {
					writeError(w, http.StatusConflict, "ese banco ya está registrado")
					return nil
				}
			}
			if item.ID > 0 {
				for i := range items {
					if items[i].ID == item.ID {
						items[i] = item
						if err := bancoStore.Write(items); err != nil {
							return err
						}
						auditar(r, "ACTUALIZÓ BANCO", item.Nombre, "")
						writeJSON(w, http.StatusOK, item)
						return nil
					}
				}
				writeError(w, http.StatusNotFound, "el banco no existe")
				return nil
			}
			item.ID = siguienteID(len(items), func(i int) int { return items[i].ID })
			items = append(items, item)
			if err := bancoStore.Write(items); err != nil {
				return err
			}
			auditar(r, "CREÓ BANCO", item.Nombre, "")
			writeJSON(w, http.StatusCreated, item)
			return nil
		})
		if err != nil {
			responderError(w, err)
		}
	case http.MethodDelete:
		id := queryInt(r, "id", 0)
		if id <= 0 {
			writeError(w, http.StatusBadRequest, "indica el banco a eliminar")
			return
		}
		err := enTransaccion(func() error {
			cuentas, err := cuentaStore.Read()
			if err != nil {
				return err
			}
			for _, c := range cuentas {
				if c.BancoID == id {
					writeError(w, http.StatusConflict, "el banco tiene cuentas asociadas; elimínalas primero")
					return nil
				}
			}
			items, err := bancoStore.Read()
			if err != nil {
				return err
			}
			for i := range items {
				if items[i].ID == id {
					eliminado := items[i]
					items = append(items[:i], items[i+1:]...)
					if err := bancoStore.Write(items); err != nil {
						return err
					}
					auditar(r, "ELIMINÓ BANCO", eliminado.Nombre, "")
					writeJSON(w, http.StatusOK, eliminado)
					return nil
				}
			}
			writeError(w, http.StatusNotFound, "el banco no existe")
			return nil
		})
		if err != nil {
			responderError(w, err)
		}
	default:
		methodNotAllowed(w)
	}
}

// Cuentas administra las cuentas bancarias de cada cooperativa.
func Cuentas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := cuentaStore.Read()
		if err != nil {
			responderError(w, err)
			return
		}
		movimientos, err := movimientoStore.Read()
		if err != nil {
			responderError(w, err)
			return
		}
		// El saldo se devuelve siempre recalculado desde los movimientos.
		services.RecalcularSaldos(items, movimientos)

		if raw := strings.TrimSpace(r.URL.Query().Get("cooperativa_id")); raw != "" {
			id, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "cooperativa_id inválido")
				return
			}
			filtradas := make([]models.Cuenta, 0)
			for _, c := range items {
				if c.CooperativaID == id {
					filtradas = append(filtradas, c)
				}
			}
			items = filtradas
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPut:
		guardarCuenta(w, r)
	case http.MethodDelete:
		eliminarCuenta(w, r)
	default:
		methodNotAllowed(w)
	}
}

func guardarCuenta(w http.ResponseWriter, r *http.Request) {
	var item models.Cuenta
	if err := readJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	item.Numero = strings.TrimSpace(item.Numero)
	item.Nombre = strings.TrimSpace(item.Nombre)
	item.Tipo = strings.TrimSpace(item.Tipo)
	item.SaldoInicial = services.Q(item.SaldoInicial)

	if item.Numero == "" {
		writeError(w, http.StatusBadRequest, "el número de cuenta es obligatorio")
		return
	}
	if item.Nombre == "" {
		writeError(w, http.StatusBadRequest, "el nombre de la cuenta es obligatorio")
		return
	}

	err := enTransaccion(func() error {
		bancos, err := bancoStore.Read()
		if err != nil {
			return err
		}
		if _, ok := buscarBanco(bancos, item.BancoID); !ok {
			writeError(w, http.StatusBadRequest, "selecciona un banco válido")
			return nil
		}
		cooperativas, err := cooperativaStore.Read()
		if err != nil {
			return err
		}
		if _, ok := buscarCooperativa(cooperativas, item.CooperativaID); !ok {
			writeError(w, http.StatusBadRequest, "selecciona una cooperativa válida")
			return nil
		}

		items, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		for _, c := range items {
			if c.ID != item.ID && c.Numero == item.Numero && c.BancoID == item.BancoID {
				writeError(w, http.StatusConflict, "esa cuenta ya existe en el banco seleccionado")
				return nil
			}
		}

		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}

		if item.ID > 0 {
			for i := range items {
				if items[i].ID != item.ID {
					continue
				}
				// Cambiar el saldo inicial reescribe todo el saldo corrido, así
				// que solo se permite mientras la cuenta no tenga movimientos.
				if !services.Iguales(items[i].SaldoInicial, item.SaldoInicial) && tieneMovimientos(movimientos, item.ID) {
					writeError(w, http.StatusConflict, "la cuenta ya tiene movimientos: registra un ajuste en lugar de cambiar el saldo inicial")
					return nil
				}
				items[i] = item
				items[i].SaldoActual = services.SaldoDeCuenta(items[i], movimientos)
				if err := cuentaStore.Write(items); err != nil {
					return err
				}
				auditar(r, "ACTUALIZÓ CUENTA", "Cuenta "+item.Numero, item.Nombre)
				writeJSON(w, http.StatusOK, items[i])
				return nil
			}
			writeError(w, http.StatusNotFound, "la cuenta no existe")
			return nil
		}

		item.ID = siguienteID(len(items), func(i int) int { return items[i].ID })
		item.SaldoActual = item.SaldoInicial
		items = append(items, item)
		if err := cuentaStore.Write(items); err != nil {
			return err
		}
		auditar(r, "CREÓ CUENTA", "Cuenta "+item.Numero, item.Nombre)
		writeJSON(w, http.StatusCreated, item)
		return nil
	})
	if err != nil {
		responderError(w, err)
	}
}

func eliminarCuenta(w http.ResponseWriter, r *http.Request) {
	id := queryInt(r, "id", 0)
	if id <= 0 {
		writeError(w, http.StatusBadRequest, "indica la cuenta a eliminar")
		return
	}
	err := enTransaccion(func() error {
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}
		if tieneMovimientos(movimientos, id) {
			writeError(w, http.StatusConflict, "la cuenta tiene movimientos registrados y no puede eliminarse")
			return nil
		}
		items, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		for i := range items {
			if items[i].ID == id {
				eliminada := items[i]
				items = append(items[:i], items[i+1:]...)
				if err := cuentaStore.Write(items); err != nil {
					return err
				}
				auditar(r, "ELIMINÓ CUENTA", "Cuenta "+eliminada.Numero, eliminada.Nombre)
				writeJSON(w, http.StatusOK, eliminada)
				return nil
			}
		}
		writeError(w, http.StatusNotFound, "la cuenta no existe")
		return nil
	})
	if err != nil {
		responderError(w, err)
	}
}

func tieneMovimientos(movimientos []models.Movimiento, cuentaID int) bool {
	for _, m := range movimientos {
		if m.CuentaID == cuentaID {
			return true
		}
	}
	return false
}

func siguienteID(total int, id func(int) int) int {
	max := 0
	for i := 0; i < total; i++ {
		if v := id(i); v > max {
			max = v
		}
	}
	return max + 1
}

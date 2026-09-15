package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, mensaje string) {
	writeJSON(w, status, map[string]string{"error": mensaje})
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "método no permitido")
}

func queryInt(r *http.Request, key string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return n
}

func queryFloat(r *http.Request, key string) (float64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

// auditar deja constancia de la acción con el usuario de la sesión.
func auditar(r *http.Request, accion, documento, detalle string) {
	usuario, _ := SesionDe(r)
	items, err := auditoriaStore.Read()
	if err != nil {
		return
	}
	registro := services.RegistrarAuditoria(items, usuario.Usuario, accion, documento, detalle, time.Now().UTC())
	registro.Rol = usuario.Rol
	items = append(items, registro)
	_ = auditoriaStore.Write(items)
}

func buscarCuenta(items []models.Cuenta, id int) (models.Cuenta, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Cuenta{}, false
}

func buscarCooperativa(items []models.Cooperativa, id int) (models.Cooperativa, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Cooperativa{}, false
}

func buscarBanco(items []models.Banco, id int) (models.Banco, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Banco{}, false
}

// contextoCuenta resuelve la cuenta con su cooperativa y banco en un solo paso.
func contextoCuenta(cuentaID int) (models.Cuenta, models.Cooperativa, models.Banco, []models.Movimiento, error) {
	cuentas, err := cuentaStore.Read()
	if err != nil {
		return models.Cuenta{}, models.Cooperativa{}, models.Banco{}, nil, err
	}
	cuenta, ok := buscarCuenta(cuentas, cuentaID)
	if !ok {
		return models.Cuenta{}, models.Cooperativa{}, models.Banco{}, nil, errNoEncontrado("la cuenta seleccionada no existe")
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		return models.Cuenta{}, models.Cooperativa{}, models.Banco{}, nil, err
	}
	cooperativa, _ := buscarCooperativa(cooperativas, cuenta.CooperativaID)
	bancos, err := bancoStore.Read()
	if err != nil {
		return models.Cuenta{}, models.Cooperativa{}, models.Banco{}, nil, err
	}
	banco, _ := buscarBanco(bancos, cuenta.BancoID)
	movimientos, err := movimientoStore.Read()
	if err != nil {
		return models.Cuenta{}, models.Cooperativa{}, models.Banco{}, nil, err
	}
	return cuenta, cooperativa, banco, movimientos, nil
}

type noEncontrado struct{ mensaje string }

func (e noEncontrado) Error() string { return e.mensaje }

func errNoEncontrado(mensaje string) error { return noEncontrado{mensaje} }

func responderError(w http.ResponseWriter, err error) {
	if _, ok := err.(noEncontrado); ok {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

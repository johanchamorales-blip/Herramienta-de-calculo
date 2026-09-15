package handlers

import (
	"net/http"
	"strings"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func Busqueda(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "debes indicar un texto de búsqueda"})
		return
	}
	coops, err := cooperativaStore.Read()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	cuentas, err := cuentaStore.Read()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	movs, err := movimientoStore.Read()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	concs, err := conciliacionStore.Read()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	resultado := services.Buscar(q, coops, bancos, cuentas, movs, concs)
	writeJSON(w, http.StatusOK, resultado)
}

var _ = models.Cooperativa{}

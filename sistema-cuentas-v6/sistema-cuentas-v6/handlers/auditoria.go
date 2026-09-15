package handlers

import (
	"net/http"
	"strings"

	"sistema-cuentas/models"
)

func Auditoria(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, err := auditoriaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	usuario := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("usuario")))
	texto := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	limite := queryInt(r, "limite", 200)
	if limite <= 0 || limite > 1000 {
		limite = 200
	}

	filtrados := make([]models.Auditoria, 0, len(items))
	for _, a := range items {
		if usuario != "" && !strings.EqualFold(a.Usuario, usuario) {
			continue
		}
		if texto != "" {
			completo := strings.ToLower(a.Accion + " " + a.DocumentoAfectado + " " + a.Detalle + " " + a.Usuario)
			if !strings.Contains(completo, texto) {
				continue
			}
		}
		filtrados = append(filtrados, a)
	}
	// Del más reciente al más antiguo.
	for i, j := 0, len(filtrados)-1; i < j; i, j = i+1, j-1 {
		filtrados[i], filtrados[j] = filtrados[j], filtrados[i]
	}
	if len(filtrados) > limite {
		filtrados = filtrados[:limite]
	}
	writeJSON(w, http.StatusOK, filtrados)
}

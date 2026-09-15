package services

import (
	"errors"
	"strings"

	"sistema-cuentas/models"
)

var ErrNotFound = errors.New("registro no encontrado")

func NextID[T interface{ GetID() int }](items []T) int {
	max := 0
	for _, item := range items {
		if id := item.GetID(); id > max {
			max = id
		}
	}
	return max + 1
}

func ValidateRequired(values ...string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return errors.New("hay campos obligatorios vacíos")
		}
	}
	return nil
}

func ValidateTipoMovimiento(m models.Movimiento) error {
	m.Tipo = strings.ToUpper(strings.TrimSpace(m.Tipo))
	if m.Tipo != "INGRESO" && m.Tipo != "EGRESO" {
		return errors.New("el tipo debe ser INGRESO o EGRESO")
	}
	return nil
}

package services

import (
	"fmt"
	"strings"
	"time"

	"sistema-cuentas/models"
)

func RegistrarAuditoria(items []models.Auditoria, usuario, accion, documento, detalle string, now time.Time) models.Auditoria {
	maxID := 0
	for _, item := range items {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	if strings.TrimSpace(usuario) == "" {
		usuario = "sistema"
	}
	return models.Auditoria{
		ID:                maxID + 1,
		Usuario:           strings.TrimSpace(usuario),
		FechaHora:         now,
		Accion:            strings.TrimSpace(accion),
		DocumentoAfectado: strings.TrimSpace(documento),
		Detalle:           strings.TrimSpace(detalle),
	}
}

func DocumentoMovimiento(m models.Movimiento, cuentaNumero string) string {
	ref := m.NumeroDocumento
	if ref == "" {
		ref = fmt.Sprintf("Movimiento #%d", m.ID)
	}
	return fmt.Sprintf("%s %s de la cuenta %s", strings.ToLower(m.Tipo), ref, cuentaNumero)
}

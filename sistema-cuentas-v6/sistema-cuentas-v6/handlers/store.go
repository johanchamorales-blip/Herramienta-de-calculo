package handlers

import (
	"path/filepath"
	"sync"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func ruta(nombre string) string { return filepath.Join("data", nombre) }

var (
	cooperativaStore  = services.JSONStore[models.Cooperativa]{Path: ruta("cooperativas.json")}
	bancoStore        = services.JSONStore[models.Banco]{Path: ruta("bancos.json")}
	cuentaStore       = services.JSONStore[models.Cuenta]{Path: ruta("cuentas.json")}
	movimientoStore   = services.JSONStore[models.Movimiento]{Path: ruta("movimientos.json")}
	conciliacionStore = services.JSONStore[models.Conciliacion]{Path: ruta("conciliaciones.json")}
	auditoriaStore    = services.JSONStore[models.Auditoria]{Path: ruta("auditoria.json")}
	usuarioStore      = services.JSONStore[models.Usuario]{Path: ruta("usuarios.json")}
)

// transaccion serializa las operaciones que leen y escriben varios archivos.
// Sin este candado, dos registros simultáneos pueden leer la misma lista, a
// asignar el mismo ID y sobrescribirse entre sí.
var transaccion sync.Mutex

func enTransaccion(fn func() error) error {
	transaccion.Lock()
	defer transaccion.Unlock()
	return fn()
}

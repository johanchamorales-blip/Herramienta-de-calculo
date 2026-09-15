package services

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
)

// Aviso es una advertencia que no impide guardar el movimiento pero que el
// usuario debe confirmar (saltos de correlativo, sobregiro, fechas atípicas).
type Aviso struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

type ResultadoMovimiento struct {
	Movimiento models.Movimiento `json:"movimiento"`
	Avisos     []Aviso           `json:"avisos"`
}

// CrearMovimiento valida y prepara un movimiento sin escribirlo todavía.
// El saldo que registra proviene del recálculo sobre los movimientos vigentes,
// no del saldo guardado en la cuenta.
func CrearMovimiento(
	m models.Movimiento,
	cuentas []models.Cuenta,
	movimientos []models.Movimiento,
	now time.Time,
) (ResultadoMovimiento, error) {
	m.Tipo = strings.ToUpper(strings.TrimSpace(m.Tipo))
	m.NumeroDocumento = strings.TrimSpace(m.NumeroDocumento)
	m.Remitente = strings.TrimSpace(m.Remitente)
	m.Beneficiario = strings.TrimSpace(m.Beneficiario)
	m.Concepto = strings.TrimSpace(m.Concepto)
	m.Monto = Q(m.Monto)

	if m.Tipo != models.TipoIngreso && m.Tipo != models.TipoEgreso {
		return ResultadoMovimiento{}, errors.New("el tipo debe ser INGRESO o EGRESO")
	}
	if m.Monto <= 0 {
		return ResultadoMovimiento{}, errors.New("el monto debe ser mayor que cero")
	}
	if m.Concepto == "" {
		return ResultadoMovimiento{}, errors.New("el concepto es obligatorio")
	}

	idx := indiceCuenta(cuentas, m.CuentaID)
	if idx == -1 {
		return ResultadoMovimiento{}, errors.New("la cuenta seleccionada no existe")
	}

	if m.FechaRegistro.IsZero() {
		m.FechaRegistro = now
	}
	hoy := models.FechaDesde(now)
	if m.FechaOperacion.EsVacia() {
		m.FechaOperacion = hoy
	}
	if m.FechaOperacion.Despues(hoy) {
		return ResultadoMovimiento{}, errors.New("la fecha de operación no puede ser futura")
	}

	avisos := make([]Aviso, 0)

	switch m.Tipo {
	case models.TipoEgreso:
		if m.Beneficiario == "" {
			return ResultadoMovimiento{}, errors.New("el beneficiario es obligatorio en un egreso")
		}
		if m.NumeroDocumento == "" {
			return ResultadoMovimiento{}, errors.New("el número de cheque o documento es obligatorio en un egreso")
		}
		m.Remitente = ""
		if m.FechaEmisionCheque.EsVacia() {
			m.FechaEmisionCheque = m.FechaOperacion
		}
		if m.FechaOperacion.Antes(m.FechaEmisionCheque) {
			return ResultadoMovimiento{}, errors.New("la fecha de operación no puede ser anterior a la fecha de emisión del cheque")
		}
		m.Estado = models.EstadoEgresoEmitido
	case models.TipoIngreso:
		if m.Remitente == "" {
			return ResultadoMovimiento{}, errors.New("el remitente es obligatorio en un ingreso")
		}
		m.Beneficiario = ""
		if !m.FechaEmisionCheque.EsVacia() && m.FechaOperacion.Antes(m.FechaEmisionCheque) {
			return ResultadoMovimiento{}, errors.New("la fecha del depósito no puede ser anterior a la fecha de emisión del cheque")
		}
		m.Estado = models.EstadoIngresoActivo
	}

	if err := validarDocumentoUnico(m, movimientos); err != nil {
		return ResultadoMovimiento{}, err
	}

	if aviso, ok := avisoSecuenciaCheque(m, movimientos); ok {
		avisos = append(avisos, aviso)
	}
	if aviso, ok := avisoFechaRetroactiva(m, movimientos); ok {
		avisos = append(avisos, aviso)
	}

	saldoPrevio := SaldoDeCuenta(cuentas[idx], movimientos)
	m.ID = nextMovimientoID(movimientos)
	m.SaldoInicial = saldoPrevio
	m.SaldoActual = saldoDespuesMovimiento(saldoPrevio, m)

	// Un sobregiro es un hecho contable posible (comisiones, notas de débito):
	// se avisa, no se bloquea, para que el libro refleje la realidad del banco.
	if m.SaldoActual < 0 {
		avisos = append(avisos, Aviso{
			Codigo:  "SOBREGIRO",
			Mensaje: fmt.Sprintf("la cuenta queda sobregirada en %s", FormatearQuetzales(-m.SaldoActual)),
		})
	}

	return ResultadoMovimiento{Movimiento: m, Avisos: avisos}, nil
}

// MarcarChequeCobrado registra que el banco pagó el cheque. Mientras un egreso
// siga en EMITIDO se considera cheque en circulación en la conciliación.
func MarcarChequeCobrado(movimientos []models.Movimiento, id int, fechaCobro models.Fecha, now time.Time) (models.Movimiento, error) {
	idx := indiceMovimiento(movimientos, id)
	if idx == -1 {
		return models.Movimiento{}, ErrNotFound
	}
	m := movimientos[idx]
	if m.Tipo != models.TipoEgreso {
		return models.Movimiento{}, errors.New("solo los egresos se marcan como cobrados")
	}
	if m.Anulado() {
		return models.Movimiento{}, errors.New("el movimiento está anulado")
	}
	if m.Estado == models.EstadoEgresoCobrado {
		return models.Movimiento{}, errors.New("el cheque ya está marcado como cobrado")
	}
	if fechaCobro.EsVacia() {
		fechaCobro = models.FechaDesde(now)
	}
	if fechaCobro.Antes(m.FechaOperacion) {
		return models.Movimiento{}, errors.New("la fecha de cobro no puede ser anterior a la fecha del cheque")
	}
	if fechaCobro.Despues(models.FechaDesde(now)) {
		return models.Movimiento{}, errors.New("la fecha de cobro no puede ser futura")
	}

	m.Estado = models.EstadoEgresoCobrado
	m.FechaCobro = fechaCobro
	movimientos[idx] = m
	return m, nil
}

// AnularMovimiento aplica borrado lógico. El saldo no se toca aquí: se recalcula
// después sobre la lista completa de movimientos.
func AnularMovimiento(movimientos []models.Movimiento, id int, motivo, usuario string, now time.Time) (models.Movimiento, error) {
	idx := indiceMovimiento(movimientos, id)
	if idx == -1 {
		return models.Movimiento{}, ErrNotFound
	}

	m := movimientos[idx]
	if m.Anulado() {
		return models.Movimiento{}, errors.New("el movimiento ya está anulado")
	}
	if m.Tipo == models.TipoEgreso && m.Estado == models.EstadoEgresoCobrado {
		return models.Movimiento{}, errors.New("un cheque ya cobrado por el banco no puede anularse; registra una nota de crédito")
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return models.Movimiento{}, errors.New("indica el motivo de la anulación")
	}

	switch m.Tipo {
	case models.TipoIngreso:
		m.Estado = models.EstadoIngresoAnulado
	case models.TipoEgreso:
		m.Estado = models.EstadoEgresoAnulado
	default:
		return models.Movimiento{}, errors.New("tipo de movimiento inválido")
	}
	m.MotivoAnulacion = motivo
	m.AnuladoPor = usuario
	m.FechaAnulacion = now
	movimientos[idx] = m
	return m, nil
}

func validarDocumentoUnico(m models.Movimiento, movimientos []models.Movimiento) error {
	if m.NumeroDocumento == "" {
		return nil
	}
	clave := strings.ToUpper(m.NumeroDocumento)
	for _, existente := range movimientos {
		if existente.ID == m.ID || existente.CuentaID != m.CuentaID || existente.Anulado() {
			continue
		}
		if existente.Tipo != m.Tipo {
			continue
		}
		if strings.ToUpper(existente.NumeroDocumento) == clave {
			return fmt.Errorf("el documento %s ya está registrado en esta cuenta (movimiento #%d)", m.NumeroDocumento, existente.ID)
		}
	}
	return nil
}

// avisoSecuenciaCheque detecta y enumera los correlativos faltantes.
func avisoSecuenciaCheque(m models.Movimiento, movimientos []models.Movimiento) (Aviso, bool) {
	if m.Tipo != models.TipoEgreso || m.NumeroDocumento == "" {
		return Aviso{}, false
	}
	nuevo, ok := numeroDocumento(m.NumeroDocumento)
	if !ok {
		return Aviso{}, false
	}

	emitidos := make([]int, 0)
	for _, x := range movimientos {
		if x.CuentaID != m.CuentaID || x.Tipo != models.TipoEgreso || x.Anulado() {
			continue
		}
		if n, ok := numeroDocumento(x.NumeroDocumento); ok {
			emitidos = append(emitidos, n)
		}
	}
	if len(emitidos) == 0 {
		return Aviso{}, false
	}

	sort.Ints(emitidos)
	ultimo := emitidos[len(emitidos)-1]
	if nuevo <= ultimo+1 {
		return Aviso{}, false
	}

	faltantes := make([]string, 0, 10)
	for n := ultimo + 1; n < nuevo && len(faltantes) < 10; n++ {
		faltantes = append(faltantes, strconv.Itoa(n))
	}
	detalle := strings.Join(faltantes, ", ")
	if nuevo-ultimo-1 > len(faltantes) {
		detalle += ", ..."
	}
	return Aviso{
		Codigo:  "SALTO_CORRELATIVO",
		Mensaje: fmt.Sprintf("quedan cheques sin registrar entre el %d y el %d: %s", ultimo, nuevo, detalle),
	}, true
}

// avisoFechaRetroactiva advierte cuando el movimiento se intercala antes del
// último registrado, porque eso reordena el saldo corrido del libro.
func avisoFechaRetroactiva(m models.Movimiento, movimientos []models.Movimiento) (Aviso, bool) {
	var ultima models.Fecha
	for _, x := range movimientos {
		if x.CuentaID != m.CuentaID || x.Anulado() {
			continue
		}
		if x.FechaOperacion.Despues(ultima) {
			ultima = x.FechaOperacion
		}
	}
	if ultima.EsVacia() || !m.FechaOperacion.Antes(ultima) {
		return Aviso{}, false
	}
	return Aviso{
		Codigo:  "FECHA_RETROACTIVA",
		Mensaje: fmt.Sprintf("la fecha %s es anterior al último movimiento (%s); el saldo corrido se reordenará", m.FechaOperacion, ultima),
	}, true
}

func FormatearQuetzales(valor float64) string {
	return fmt.Sprintf("Q %.2f", Q(valor))
}

func numeroDocumento(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	digitos := strings.Builder{}
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digitos.WriteRune(r)
		}
	}
	if digitos.Len() == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(digitos.String())
	if err != nil {
		return 0, false
	}
	return n, true
}

func indiceCuenta(cuentas []models.Cuenta, id int) int {
	for i := range cuentas {
		if cuentas[i].ID == id {
			return i
		}
	}
	return -1
}

func indiceMovimiento(movimientos []models.Movimiento, id int) int {
	for i := range movimientos {
		if movimientos[i].ID == id {
			return i
		}
	}
	return -1
}

func nextMovimientoID(items []models.Movimiento) int {
	max := 0
	for _, x := range items {
		if x.ID > max {
			max = x.ID
		}
	}
	return max + 1
}

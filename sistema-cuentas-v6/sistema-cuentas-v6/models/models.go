package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	TipoIngreso = "INGRESO"
	TipoEgreso  = "EGRESO"
	EstadoIngresoActivo = "ACTIVO"
	EstadoIngresoAnulado = "ANULADO"
	EstadoEgresoEmitido = "EMITIDO"
	EstadoEgresoCobrado = "COBRADO"
	EstadoEgresoAnulado = "ANULADO"
	RolAdministrador = "ADMINISTRADOR"
	RolContador = "CONTADOR"
	RolJefePlanta = "JEFE_DE_PLANTA"
)

type Fecha struct { time.Time }
const formatoFecha = "2006-01-02"
func NuevaFecha(anio int, mes time.Month, dia int) Fecha { return Fecha{time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)} }
func FechaDesde(t time.Time) Fecha { return NuevaFecha(t.Year(), t.Month(), t.Day()) }
func ParsearFecha(valor string) (Fecha, error) { valor = strings.TrimSpace(valor); if valor == "" { return Fecha{}, nil }; if t, err := time.Parse(formatoFecha, valor); err == nil { return Fecha{t}, nil }; if t, err := time.Parse(time.RFC3339, valor); err == nil { return FechaDesde(t), nil }; return Fecha{}, errors.New("fecha inválida, usa el formato AAAA-MM-DD") }
func (f Fecha) EsVacia() bool { return f.Time.IsZero() }
func (f Fecha) String() string { if f.EsVacia() { return "" }; return f.Time.Format(formatoFecha) }
func (f Fecha) Antes(otra Fecha) bool { return f.Time.Before(otra.Time) }
func (f Fecha) Despues(otra Fecha) bool { return f.Time.After(otra.Time) }
func (f Fecha) EnPeriodo(mes, anio int) bool { return !f.EsVacia() && int(f.Time.Month()) == mes && f.Time.Year() == anio }
func (f Fecha) MarshalJSON() ([]byte, error) { if f.EsVacia() { return []byte(`""`), nil }; return json.Marshal(f.String()) }
func (f *Fecha) UnmarshalJSON(data []byte) error { var valor any; if err := json.Unmarshal(data, &valor); err != nil { return err }; switch v := valor.(type) { case nil: *f = Fecha{}; return nil; case string: parsed, err := ParsearFecha(v); if err != nil { return err }; *f = parsed; return nil; default: return errors.New("fecha inválida, usa el formato AAAA-MM-DD") } }

type Usuario struct { ID int `json:"id"`; Usuario string `json:"usuario"`; Email string `json:"email"`; Rol string `json:"rol"`; HashPassword string `json:"hash_password"`; Salt string `json:"salt"`; Activo bool `json:"activo"`; FechaCreacion time.Time `json:"fecha_creacion"`; UltimoAcceso time.Time `json:"ultimo_acceso,omitempty"` }
func (u Usuario) Publico() map[string]any { return map[string]any{"id":u.ID,"usuario":u.Usuario,"email":u.Email,"rol":u.Rol,"activo":u.Activo} }
type Cooperativa struct { ID int `json:"id"`; Nombre string `json:"nombre"`; Direccion string `json:"direccion"`; NIT string `json:"nit"`; Telefono string `json:"telefono"` }
type Banco struct { ID int `json:"id"`; Nombre string `json:"nombre"` }
type Cuenta struct { ID int `json:"id"`; BancoID int `json:"banco_id"`; CooperativaID int `json:"cooperativa_id"`; Nombre string `json:"nombre"`; Numero string `json:"numero"`; Tipo string `json:"tipo"`; SaldoInicial float64 `json:"saldo_inicial"`; SaldoActual float64 `json:"saldo_actual"` }

type MovimientoBanco struct { Fecha Fecha `json:"fecha"`; NumeroDocumento string `json:"numero_documento"`; Descripcion string `json:"descripcion"`; Monto float64 `json:"monto"`; Tipo string `json:"tipo"`; Debito float64 `json:"debito"`; Credito float64 `json:"credito"`; Saldo float64 `json:"saldo,omitempty"`; TieneSaldo bool `json:"tiene_saldo,omitempty"`; Conciliado bool `json:"conciliado"`; MovimientoLibroID int `json:"movimiento_libro_id,omitempty"`; Coincidencia string `json:"coincidencia,omitempty"` }

type Conciliacion struct { ID int `json:"id"`; CuentaID int `json:"cuenta_id"`; Mes int `json:"mes"`; Anio int `json:"anio"`; SaldoLibros float64 `json:"saldo_libros"`; SaldoLibrosAjustado float64 `json:"saldo_libros_ajustado"`; SaldoEstadoCuenta float64 `json:"saldo_estado_cuenta"`; ChequesCirculacion float64 `json:"cheques_circulacion"`; DepositosEnTransito float64 `json:"depositos_en_transito"`; NotasDebito float64 `json:"notas_debito"`; NotasCredito float64 `json:"notas_credito"`; Ajustes float64 `json:"ajustes"`; SubtotalBanco float64 `json:"subtotal_banco"`; SaldoConciliado float64 `json:"saldo_conciliado"`; DiferenciaConLibros float64 `json:"diferencia_con_libros"`; FechaLugar string `json:"fecha_lugar"`; Observaciones string `json:"observaciones,omitempty"`; Usuario string `json:"usuario,omitempty"`; FechaCreacion time.Time `json:"fecha_creacion"`; MovimientosBanco []MovimientoBanco `json:"movimientos_banco,omitempty"` }

type Movimiento struct { ID int `json:"id"`; CuentaID int `json:"cuenta_id"`; Tipo string `json:"tipo"`; FechaRegistro time.Time `json:"fecha_registro"`; FechaOperacion Fecha `json:"fecha_operacion"`; FechaEmisionCheque Fecha `json:"fecha_emision_cheque,omitempty"`; FechaCobro Fecha `json:"fecha_cobro,omitempty"`; NumeroDocumento string `json:"numero_documento"`; Remitente string `json:"remitente,omitempty"`; Beneficiario string `json:"beneficiario,omitempty"`; Concepto string `json:"concepto"`; Monto float64 `json:"monto"`; Estado string `json:"estado"`; SaldoInicial float64 `json:"saldo_inicial"`; SaldoActual float64 `json:"saldo_actual"`; Usuario string `json:"usuario,omitempty"`; MotivoAnulacion string `json:"motivo_anulacion,omitempty"`; AnuladoPor string `json:"anulado_por,omitempty"`; FechaAnulacion time.Time `json:"fecha_anulacion,omitempty"` }
func (m Movimiento) Anulado() bool { return m.Estado == EstadoIngresoAnulado || m.Estado == EstadoEgresoAnulado }
func (m Movimiento) Afecta() bool { return !m.Anulado() }
type Auditoria struct { ID int `json:"id"`; Usuario string `json:"usuario"`; Rol string `json:"rol,omitempty"`; FechaHora time.Time `json:"fecha_hora"`; Accion string `json:"accion"`; DocumentoAfectado string `json:"documento_afectado"`; Detalle string `json:"detalle,omitempty"` }

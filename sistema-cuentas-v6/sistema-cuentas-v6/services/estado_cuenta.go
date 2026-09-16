package services

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"sistema-cuentas/models"
)

type ResultadoImportacionEstadoCuenta struct {
	NombreArchivo          string                    `json:"nombre_archivo"`
	Mes                    int                       `json:"mes"`
	Anio                   int                       `json:"anio"`
	SaldoEstadoCuenta     float64                   `json:"saldo_estado_cuenta"`
	DepositosEnTransito   float64                   `json:"depositos_en_transito"`
	NotasDebito            float64                   `json:"notas_debito"`
	NotasCredito           float64                   `json:"notas_credito"`
	Ajustes                 float64                   `json:"ajustes"`
	MovimientosBanco       []models.MovimientoBanco `json:"movimientos_banco"`
	MovimientosDetectados int                       `json:"movimientos_detectados"`
	MovimientosConciliados int                      `json:"movimientos_conciliados"`
	MovimientosPendientes  int                      `json:"movimientos_pendientes"`
	Advertencias            []string                 `json:"advertencias,omitempty"`
}

type columnaDetectada struct {
	Indice int
	Tipo   string
}

var nombresColumna = map[string][]string{
	"fecha":       {"fecha", "date", "fechaoperacion", "transactiondate", "postingdate"},
	"documento":   {"cheque", "numero", "documento", "referencia", "checknumber", "numerodocumento", "reference"},
	"descripcion": {"descripcion", "concepto", "detalle", "glosa", "description", "memo"},
	"debito":      {"debito", "debitos", "retiro", "retiros", "cargo", "cargos", "debit", "withdrawal"},
	"credito":     {"credito", "creditos", "abono", "abonos", "deposito", "depositos", "credit", "deposit"},
	"monto":       {"monto", "importe", "amount", "valor", "transactionamount"},
	"saldo":       {"saldo", "balance", "saldofinal", "runningbalance"},
}

func ImportarEstadoCuenta(nombre string, data []byte) (ResultadoImportacionEstadoCuenta, error) {
	ext := strings.ToLower(filepath.Ext(nombre))
	var filas [][]string
	var err error
	switch ext {
	case ".csv":
		filas, err = leerCSV(data)
	case ".xlsx":
		filas, err = leerXLSX(data)
	default:
		return ResultadoImportacionEstadoCuenta{}, errors.New("por ahora el estado de cuenta debe ser CSV o Excel .xlsx")
	}
	if err != nil {
		return ResultadoImportacionEstadoCuenta{}, err
	}
	return interpretarFilas(nombre, filas)
}

func CompararMovimientosBanco(bank []models.MovimientoBanco, libros []models.Movimiento, mes, anio int) (float64, float64, float64) {
	usados := make(map[int]bool)
	for i := range bank {
		mejor := -1
		mejorPuntaje := 0
		for j, m := range libros {
			if usados[m.ID] || m.Anulado() || !m.FechaOperacion.EnPeriodo(mes, anio) || m.Tipo != bank[i].Tipo || !Iguales(m.Monto, bank[i].Monto) {
				continue
			}
			puntaje := 1
			if bank[i].NumeroDocumento != "" && m.NumeroDocumento != "" && normalizarClave(bank[i].NumeroDocumento) == normalizarClave(m.NumeroDocumento) {
				puntaje = 3
			} else if !bank[i].Fecha.EsVacia() && absDias(bank[i].Fecha.Time, m.FechaOperacion.Time) <= 7 {
				puntaje = 2
			}
			if puntaje > mejorPuntaje {
				mejorPuntaje = puntaje
				mejor = j
			}
		}
		if mejor >= 0 {
			m := libros[mejor]
			usados[m.ID] = true
			bank[i].Conciliado = true
			bank[i].MovimientoLibroID = m.ID
			switch mejorPuntaje {
			case 3:
				bank[i].Coincidencia = "Documento + monto"
			case 2:
				bank[i].Coincidencia = "Monto + fecha"
			default:
				bank[i].Coincidencia = "Monto"
			}
		}
	}

	var depositos, debitos, creditos float64
	for _, b := range bank {
		texto := strings.ToLower(b.Descripcion)
		if strings.Contains(texto, "nota de debito") || strings.Contains(texto, "nota debito") {
			debitos = Suma(debitos, b.Monto)
		}
		if strings.Contains(texto, "nota de credito") || strings.Contains(texto, "nota credito") {
			creditos = Suma(creditos, b.Monto)
		}
	}
	for _, m := range libros {
		if m.Anulado() || m.CuentaID <= 0 || !m.FechaOperacion.EnPeriodo(mes, anio) || m.Tipo != models.TipoIngreso || usados[m.ID] {
			continue
		}
		depositos = Suma(depositos, m.Monto)
	}
	return depositos, debitos, creditos
}

func interpretarFilas(nombre string, filas [][]string) (ResultadoImportacionEstadoCuenta, error) {
	if len(filas) < 2 {
		return ResultadoImportacionEstadoCuenta{}, errors.New("el archivo no contiene suficientes filas para detectar un estado de cuenta")
	}
	header, indice := detectarEncabezado(filas)
	if indice < 0 {
		return ResultadoImportacionEstadoCuenta{}, errors.New("no pude identificar las columnas del estado de cuenta; usa encabezados como Fecha, Descripción, Débito, Crédito y Saldo")
	}
	columnas := detectarColumnas(header)
	if columnas["fecha"].Indice < 0 {
		return ResultadoImportacionEstadoCuenta{}, errors.New("falta una columna de fecha")
	}
	resultado := ResultadoImportacionEstadoCuenta{NombreArchivo: nombre, MovimientosBanco: []models.MovimientoBanco{}, Advertencias: []string{}}
	maxFecha := models.Fecha{}
	for _, fila := range filas[indice+1:] {
		if filaVacia(fila) {
			continue
		}
		fecha, ok := leerFecha(celda(fila, columnas["fecha"].Indice))
		if !ok {
			continue
		}
		debito, tieneDebito := leerMonto(celda(fila, columnas["debito"].Indice))
		credito, tieneCredito := leerMonto(celda(fila, columnas["credito"].Indice))
		monto, tieneMonto := leerMonto(celda(fila, columnas["monto"].Indice))
		if !tieneDebito && !tieneCredito && !tieneMonto {
			continue
		}
		if !tieneDebito && !tieneCredito && tieneMonto {
			if monto < 0 {
				debito = -monto
			} else {
				credito = monto
			}
		}
		if debito == 0 && credito == 0 {
			continue
		}
		tipo := models.TipoIngreso
		valor := credito
		if debito > 0 {
			tipo = models.TipoEgreso
			valor = debito
		}
		mov := models.MovimientoBanco{
			Fecha:            fecha,
			NumeroDocumento: strings.TrimSpace(celda(fila, columnas["documento"].Indice)),
			Descripcion:      strings.TrimSpace(celda(fila, columnas["descripcion"].Indice)),
			Monto:            Q(valor),
			Tipo:             tipo,
			Debito:           Q(debito),
			Credito:          Q(credito),
		}
		if saldo, ok := leerMonto(celda(fila, columnas["saldo"].Indice)); ok {
			mov.Saldo = Q(saldo)
			mov.TieneSaldo = true
			resultado.SaldoEstadoCuenta = Q(saldo)
		}
		if fecha.Despues(maxFecha) {
			maxFecha = fecha
		}
		resultado.MovimientosBanco = append(resultado.MovimientosBanco, mov)
	}
	if maxFecha.EsVacia() || len(resultado.MovimientosBanco) == 0 {
		return ResultadoImportacionEstadoCuenta{}, errors.New("no se encontraron movimientos bancarios legibles")
	}
	resultado.Mes, resultado.Anio = int(maxFecha.Time.Month()), maxFecha.Time.Year()
	if resultado.SaldoEstadoCuenta == 0 {
		resultado.Advertencias = append(resultado.Advertencias, "No se detectó una columna Saldo/Balance; el saldo deberá revisarse manualmente.")
	}
	if columnas["documento"].Indice < 0 {
		resultado.Advertencias = append(resultado.Advertencias, "No se detectó columna de cheque/referencia; la comparación usará monto y fecha.")
	}
	resultado.MovimientosDetectados = len(resultado.MovimientosBanco)
	return resultado, nil
}

func leerCSV(data []byte) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.TrimLeadingSpace = true
	muestra := bytes.ReplaceAll(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), []byte("\r\n"), []byte("\n"))
	if bytes.Contains(muestra, []byte(";")) && !bytes.Contains(muestra, []byte(",")) {
		r.Comma = ';'
	} else if bytes.Contains(muestra, []byte("\t")) && !bytes.Contains(muestra, []byte(",")) {
		r.Comma = '\t'
	}
	var filas [][]string
	for {
		fila, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("leer CSV: %w", err)
		}
		if len(filas) == 0 && len(fila) > 0 {
			fila[0] = strings.TrimPrefix(fila[0], "\ufeff")
		}
		filas = append(filas, fila)
	}
	return filas, nil
}

type xlsxCell struct {
	Ref string `xml:"r,attr"`
	Type string `xml:"t,attr"`
	V string `xml:"v"`
	Is struct { T string `xml:"t"` } `xml:"is"`
}

type xlsxRow struct { Cells []xlsxCell `xml:"c"` }
type xlsxSheet struct { Rows []xlsxRow `xml:"sheetData>row"` }
type xlsxSI struct { Text []string `xml:"t"` }
type xlsxSST struct { Items []xlsxSI `xml:"si"` }

func leerXLSX(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil { return nil, fmt.Errorf("abrir Excel: %w", err) }
	shared := []string{}
	for _, f := range zr.File {
		if f.Name != "xl/sharedStrings.xml" { continue }
		rc, e := f.Open(); if e != nil { return nil, e }
		var s xlsxSST; e = xml.NewDecoder(rc).Decode(&s); rc.Close(); if e != nil { return nil, e }
		for _, item := range s.Items { shared = append(shared, strings.Join(item.Text, "")) }
	}
	var hoja *zip.File
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") { hoja = f; break }
	}
	if hoja == nil { return nil, errors.New("el Excel no contiene una hoja legible") }
	rc, err := hoja.Open(); if err != nil { return nil, err }
	var sheet xlsxSheet
	err = xml.NewDecoder(rc).Decode(&sheet); rc.Close(); if err != nil { return nil, fmt.Errorf("leer hoja Excel: %w", err) }
	var filas [][]string
	for _, row := range sheet.Rows {
		max := 0
		tmp := map[int]string{}
		for _, c := range row.Cells {
			idx := columnaExcel(c.Ref)
			if idx < 0 { continue }
			valor := c.V
			if c.Type == "s" {
				n, _ := strconv.Atoi(valor); if n >= 0 && n < len(shared) { valor = shared[n] }
			} else if c.Type == "inlineStr" { valor = c.Is.T }
			tmp[idx] = valor; if idx > max { max = idx }
		}
		fila := make([]string, max+1)
		for idx, valor := range tmp { fila[idx] = valor }
		filas = append(filas, fila)
	}
	return filas, nil
}

func columnaExcel(ref string) int {
	ref = strings.ToUpper(ref)
	i := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' { break }
		i = i*26 + int(r-'A'+1)
	}
	return i - 1
}

func detectarEncabezado(filas [][]string) ([]string, int) {
	mejor := -1; mejorPuntaje := 0
	limite := len(filas); if limite > 20 { limite = 20 }
	for i := 0; i < limite; i++ {
		p := 0
		for _, valor := range filas[i] {
			v := normalizarCabecera(valor)
			for _, candidatos := range nombresColumna {
				for _, candidato := range candidatos { if v == normalizarCabecera(candidato) { p++; break } }
			}
		}
		if p > mejorPuntaje { mejorPuntaje, mejor = p, i }
	}
	if mejorPuntaje < 2 { return nil, -1 }
	return filas[mejor], mejor
}

func detectarColumnas(header []string) map[string]columnaDetectada {
	out := map[string]columnaDetectada{}
	for k := range nombresColumna { out[k] = columnaDetectada{Indice: -1} }
	for i, h := range header {
		v := normalizarCabecera(h)
		for tipo, candidatos := range nombresColumna {
			for _, candidato := range candidatos {
				if v == normalizarCabecera(candidato) { if out[tipo].Indice < 0 { out[tipo] = columnaDetectada{Indice:i, Tipo:tipo} }; break }
			}
		}
	}
	return out
}

func normalizarCabecera(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	reemplazos := strings.NewReplacer("á","a","é","e","í","i","ó","o","ú","u","ü","u","ñ","n")
	v = reemplazos.Replace(v)
	var b strings.Builder
	for _, r := range v { if unicode.IsLetter(r) || unicode.IsDigit(r) { b.WriteRune(r) } }
	return b.String()
}

func normalizarClave(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	var b strings.Builder
	for _, r := range v { if unicode.IsLetter(r) || unicode.IsDigit(r) { b.WriteRune(r) } }
	return b.String()
}

func leerFecha(raw string) (models.Fecha, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" { return models.Fecha{}, false }
	if n, err := strconv.ParseFloat(raw, 64); err == nil && n > 20000 && n < 70000 { return models.FechaDesde(time.Date(1899,12,30,0,0,0,0,time.UTC).Add(time.Duration(n*24)*time.Hour)), true }
	for _, formato := range []string{"2006-01-02", "02/01/2006", "01/02/2006", "02-01-2006", "2006/01/02", "02.01.2006", time.RFC3339} {
		if t, err := time.Parse(formato, raw); err == nil { return models.FechaDesde(t), true }
	}
	return models.Fecha{}, false
}

func leerMonto(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw); if raw == "" { return 0, false }
	raw = strings.ReplaceAll(raw, "Q", ""); raw = strings.ReplaceAll(raw, "$", ""); raw = strings.ReplaceAll(raw, "€", ""); raw = strings.ReplaceAll(raw, " ", "")
	neg := strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")"); raw = strings.Trim(raw, "()")
	if strings.Contains(raw, ",") && strings.Contains(raw, ".") {
		if strings.LastIndex(raw, ",") > strings.LastIndex(raw, ".") { raw = strings.ReplaceAll(raw, ".", ""); raw = strings.Replace(raw, ",", ".", 1) } else { raw = strings.ReplaceAll(raw, ",", "") }
	} else if strings.Count(raw, ",") == 1 { p := strings.LastIndex(raw, ","); if len(raw)-p-1 <= 2 { raw = strings.Replace(raw, ",", ".", 1) } else { raw = strings.ReplaceAll(raw, ",", "") } }
	raw = strings.ReplaceAll(raw, ",", "")
	n, err := strconv.ParseFloat(raw, 64); if err != nil { return 0, false }
	if neg { n = -n }
	return Q(n), true
}

func celda(fila []string, idx int) string { if idx < 0 || idx >= len(fila) { return "" }; return fila[idx] }
func filaVacia(fila []string) bool { for _, v := range fila { if strings.TrimSpace(v) != "" { return false } }; return true }
func absDias(a, b time.Time) int { d := a.Sub(b); if d < 0 { d = -d }; return int(d.Hours()/24) }

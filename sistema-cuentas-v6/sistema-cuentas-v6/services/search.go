package services

import (
	"sort"
	"strings"
	"unicode"

	"sistema-cuentas/models"
)

type ResultadoBusqueda struct {
	Cooperativas []ResultadoCooperativa        `json:"cooperativas"`
	Cuentas      []ResultadoCuenta             `json:"cuentas"`
	Movimientos  []ResultadoMovimientoBusqueda `json:"movimientos"`
}

type ResultadoCooperativa struct {
	Cooperativa models.Cooperativa `json:"cooperativa"`
	Puntaje     int                `json:"puntaje"`
	Bancos      []BancoConCuentas  `json:"bancos"`
}

type BancoConCuentas struct {
	Banco   models.Banco    `json:"banco"`
	Cuentas []CuentaDetalle `json:"cuentas"`
}

type CuentaDetalle struct {
	Cuenta            models.Cuenta `json:"cuenta"`
	LibrosMovimientos int           `json:"libros_movimientos"`
	Conciliaciones    int           `json:"conciliaciones"`
}

type ResultadoCuenta struct {
	Cuenta         models.Cuenta      `json:"cuenta"`
	Cooperativa    models.Cooperativa `json:"cooperativa"`
	Banco          models.Banco       `json:"banco"`
	Puntaje        int                `json:"puntaje"`
	Libros         int                `json:"libros"`
	Conciliaciones int                `json:"conciliaciones"`
}

type ResultadoMovimientoBusqueda struct {
	Movimiento  models.Movimiento  `json:"movimiento"`
	Cuenta      models.Cuenta      `json:"cuenta"`
	Cooperativa models.Cooperativa `json:"cooperativa"`
	Banco       models.Banco       `json:"banco"`
	Puntaje     int                `json:"puntaje"`
}

func Buscar(q string, cooperativas []models.Cooperativa, bancos []models.Banco, cuentas []models.Cuenta, movimientos []models.Movimiento, conciliaciones []models.Conciliacion) ResultadoBusqueda {
	q = normalizar(q)
	res := ResultadoBusqueda{
		Cooperativas: []ResultadoCooperativa{},
		Cuentas:      []ResultadoCuenta{},
		Movimientos:  []ResultadoMovimientoBusqueda{},
	}
	if q == "" {
		return res
	}

	for _, c := range cooperativas {
		puntaje := mejorPuntaje(q, c.Nombre, c.NIT, c.Direccion, c.Telefono)
		if puntaje <= 60 {
			continue
		}
		byBank := map[int]*BancoConCuentas{}
		for _, cuenta := range cuentas {
			if cuenta.CooperativaID != c.ID {
				continue
			}
			b, ok := buscarBancoLocal(bancos, cuenta.BancoID)
			if !ok {
				continue
			}
			entry, ok := byBank[b.ID]
			if !ok {
				entry = &BancoConCuentas{Banco: b, Cuentas: []CuentaDetalle{}}
				byBank[b.ID] = entry
			}
			libs := 0
			for _, m := range movimientos {
				if m.CuentaID == cuenta.ID && m.Afecta() {
					libs++
				}
			}
			concs := 0
			for _, con := range conciliaciones {
				if con.CuentaID == cuenta.ID {
					concs++
				}
			}
			entry.Cuentas = append(entry.Cuentas, CuentaDetalle{Cuenta: cuenta, LibrosMovimientos: libs, Conciliaciones: concs})
		}
		banks := make([]BancoConCuentas, 0, len(byBank))
		for _, b := range byBank {
			sort.Slice(b.Cuentas, func(i, j int) bool { return b.Cuentas[i].Cuenta.Numero < b.Cuentas[j].Cuenta.Numero })
			banks = append(banks, *b)
		}
		sort.Slice(banks, func(i, j int) bool { return banks[i].Banco.Nombre < banks[j].Banco.Nombre })
		res.Cooperativas = append(res.Cooperativas, ResultadoCooperativa{Cooperativa: c, Puntaje: puntaje, Bancos: banks})
	}

	for _, cuenta := range cuentas {
		b, bok := buscarBancoLocal(bancos, cuenta.BancoID)
		c, cok := buscarCooperativaLocal(cooperativas, cuenta.CooperativaID)
		if !bok || !cok {
			continue
		}
		puntaje := mejorPuntaje(q, cuenta.Numero, b.Nombre, c.Nombre, c.NIT)
		if puntaje <= 60 {
			continue
		}
		libs, concs := 0, 0
		for _, m := range movimientos {
			if m.CuentaID == cuenta.ID && m.Afecta() {
				libs++
			}
		}
		for _, con := range conciliaciones {
			if con.CuentaID == cuenta.ID {
				concs++
			}
		}
		res.Cuentas = append(res.Cuentas, ResultadoCuenta{Cuenta: cuenta, Cooperativa: c, Banco: b, Puntaje: puntaje, Libros: libs, Conciliaciones: concs})
	}

	for _, m := range movimientos {
		c, cok := buscarCuentaLocal(cuentas, m.CuentaID)
		if !cok {
			continue
		}
		coop, cok := buscarCooperativaLocal(cooperativas, c.CooperativaID)
		if !cok {
			continue
		}
		bank, bok := buscarBancoLocal(bancos, c.BancoID)
		if !bok {
			continue
		}
		puntaje := mejorPuntaje(q, m.NumeroDocumento, m.Remitente, m.Beneficiario, m.Concepto)
		if puntaje <= 70 {
			continue
		}
		res.Movimientos = append(res.Movimientos, ResultadoMovimientoBusqueda{Movimiento: m, Cuenta: c, Cooperativa: coop, Banco: bank, Puntaje: puntaje})
	}

	sort.Slice(res.Cooperativas, func(i, j int) bool { return res.Cooperativas[i].Puntaje > res.Cooperativas[j].Puntaje })
	sort.Slice(res.Cuentas, func(i, j int) bool { return res.Cuentas[i].Puntaje > res.Cuentas[j].Puntaje })
	sort.Slice(res.Movimientos, func(i, j int) bool { return res.Movimientos[i].Puntaje > res.Movimientos[j].Puntaje })
	return res
}

func normalizar(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		switch r {
		case 'á', 'à', 'ä', 'â':
			r = 'a'
		case 'é', 'è', 'ë', 'ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û':
			r = 'u'
		case 'ñ':
			r = 'n'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func mejorPuntaje(q string, values ...string) int {
	best := 0
	for _, value := range values {
		v := normalizar(value)
		if v == "" {
			continue
		}
		if v == q {
			best = 100
			continue
		}
		if strings.Contains(v, q) || strings.Contains(q, v) {
			if best < 90 {
				best = 90
			}
			continue
		}
		d := levenshtein(q, v)
		maxLen := len([]rune(q))
		if len([]rune(v)) > maxLen {
			maxLen = len([]rune(v))
		}
		if maxLen == 0 {
			continue
		}
		score := 100 - (d * 100 / maxLen)
		if score > best {
			best = score
		}
	}
	return best
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ca := range ra {
		cur[0] = i + 1
		for j, cb := range rb {
			cost := 0
			if ca != cb {
				cost = 1
			}
			cur[j+1] = min3(cur[j]+1, prev[j+1]+1, prev[j]+cost)
		}
		copy(prev, cur)
	}
	return prev[len(rb)]
}
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
func buscarBancoLocal(items []models.Banco, id int) (models.Banco, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Banco{}, false
}
func buscarCooperativaLocal(items []models.Cooperativa, id int) (models.Cooperativa, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Cooperativa{}, false
}
func buscarCuentaLocal(items []models.Cuenta, id int) (models.Cuenta, bool) {
	for _, x := range items {
		if x.ID == id {
			return x, true
		}
	}
	return models.Cuenta{}, false
}

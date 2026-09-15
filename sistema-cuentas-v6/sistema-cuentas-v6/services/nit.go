package services

import (
	"errors"
	"strings"
)

// NormalizarNIT deja el NIT en mayúsculas, sin espacios ni guiones.
func NormalizarNIT(nit string) string {
	limpio := strings.ToUpper(strings.TrimSpace(nit))
	limpio = strings.ReplaceAll(limpio, "-", "")
	limpio = strings.ReplaceAll(limpio, " ", "")
	return limpio
}

// FormatearNIT devuelve el NIT con el guion antes del dígito verificador.
func FormatearNIT(nit string) string {
	limpio := NormalizarNIT(nit)
	if len(limpio) < 2 || limpio == "CF" {
		return limpio
	}
	return limpio[:len(limpio)-1] + "-" + limpio[len(limpio)-1:]
}

// ValidarNIT verifica el dígito verificador del NIT guatemalteco (módulo 11).
// Evita que un dígito mal tecleado cree una cooperativa duplicada o imposible
// de cruzar con los documentos del banco.
func ValidarNIT(nit string) error {
	limpio := NormalizarNIT(nit)
	if limpio == "" {
		return errors.New("el NIT es obligatorio")
	}
	if limpio == "CF" {
		return errors.New("una cooperativa no puede registrarse como consumidor final")
	}
	if len(limpio) < 2 {
		return errors.New("el NIT es demasiado corto")
	}

	// Los NIT derivados del CUI (13 dígitos) no llevan dígito verificador
	// módulo 11, así que solo se valida que sean numéricos.
	if len(limpio) == 13 && soloDigitos(limpio) {
		return nil
	}

	cuerpo := limpio[:len(limpio)-1]
	verificador := limpio[len(limpio)-1:]

	for _, r := range cuerpo {
		if r < '0' || r > '9' {
			return errors.New("el NIT solo admite dígitos y un verificador final")
		}
	}

	suma := 0
	peso := len(cuerpo) + 1
	for _, r := range cuerpo {
		suma += int(r-'0') * peso
		peso--
	}

	modulo := (11 - (suma % 11)) % 11
	esperado := "K"
	if modulo != 10 {
		esperado = string(rune('0' + modulo))
	}
	if verificador != esperado {
		return errors.New("el dígito verificador del NIT no es válido")
	}
	return nil
}

func soloDigitos(valor string) bool {
	for _, r := range valor {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(valor) > 0
}

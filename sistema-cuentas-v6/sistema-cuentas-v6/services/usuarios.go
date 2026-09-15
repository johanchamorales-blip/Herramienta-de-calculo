package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"

	"sistema-cuentas/models"
)

const (
	iteracionesPBKDF2 = 120000
	largoClaveHash    = 32
)

var rolesValidos = map[string]string{
	models.RolAdministrador: "Administrador",
	models.RolContador:      "Contador",
	models.RolJefePlanta:    "Jefe de planta",
}

// permisos por rol. Lectura la tienen todos los roles autenticados.
var permisosPorRol = map[string][]string{
	models.RolAdministrador: {"catalogos", "movimientos", "conciliacion", "auditoria", "usuarios"},
	models.RolContador:      {"catalogos", "movimientos", "conciliacion", "auditoria"},
	models.RolJefePlanta:    {"auditoria"},
}

func NombreRol(rol string) string {
	if nombre, ok := rolesValidos[rol]; ok {
		return nombre
	}
	return rol
}

func RolesDisponibles() []map[string]string {
	orden := []string{models.RolAdministrador, models.RolContador, models.RolJefePlanta}
	salida := make([]map[string]string, 0, len(orden))
	for _, rol := range orden {
		salida = append(salida, map[string]string{"valor": rol, "nombre": rolesValidos[rol]})
	}
	return salida
}

func Permisos(rol string) []string {
	if p, ok := permisosPorRol[rol]; ok {
		return p
	}
	return []string{}
}

func Puede(rol, permiso string) bool {
	for _, p := range Permisos(rol) {
		if p == permiso {
			return true
		}
	}
	return false
}

// ValidarPassword exige una contraseña razonable sin volverse hostil.
func ValidarPassword(password string) error {
	if len([]rune(password)) < 8 {
		return errors.New("la contraseña necesita al menos 8 caracteres")
	}
	var tieneLetra, tieneNumero bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			tieneLetra = true
		case unicode.IsDigit(r):
			tieneNumero = true
		}
	}
	if !tieneLetra || !tieneNumero {
		return errors.New("la contraseña necesita letras y números")
	}
	return nil
}

func ValidarEmail(email string) error {
	email = strings.TrimSpace(email)
	arroba := strings.Index(email, "@")
	punto := strings.LastIndex(email, ".")
	if arroba < 1 || punto < arroba+2 || punto == len(email)-1 {
		return errors.New("el correo no tiene un formato válido")
	}
	return nil
}

// CrearUsuario valida los datos y devuelve el usuario con la contraseña
// derivada con PBKDF2-SHA256 y sal aleatoria.
func CrearUsuario(existentes []models.Usuario, usuario, email, password, rol string, now time.Time) (models.Usuario, error) {
	usuario = strings.TrimSpace(usuario)
	email = strings.ToLower(strings.TrimSpace(email))
	rol = strings.ToUpper(strings.TrimSpace(rol))

	if len(usuario) < 3 {
		return models.Usuario{}, errors.New("el usuario necesita al menos 3 caracteres")
	}
	if err := ValidarEmail(email); err != nil {
		return models.Usuario{}, err
	}
	if _, ok := rolesValidos[rol]; !ok {
		return models.Usuario{}, errors.New("selecciona un rol válido")
	}
	if err := ValidarPassword(password); err != nil {
		return models.Usuario{}, err
	}

	maxID := 0
	for _, u := range existentes {
		if strings.EqualFold(u.Usuario, usuario) {
			return models.Usuario{}, errors.New("ese usuario ya está registrado")
		}
		if strings.EqualFold(u.Email, email) {
			return models.Usuario{}, errors.New("ese correo ya está registrado")
		}
		if u.ID > maxID {
			maxID = u.ID
		}
	}

	salt, err := salAleatoria()
	if err != nil {
		return models.Usuario{}, err
	}

	// El primer usuario del sistema es administrador para que exista quien
	// pueda gestionar a los demás.
	if len(existentes) == 0 {
		rol = models.RolAdministrador
	}

	return models.Usuario{
		ID:            maxID + 1,
		Usuario:       usuario,
		Email:         email,
		Rol:           rol,
		Salt:          salt,
		HashPassword:  derivarClave(password, salt),
		Activo:        true,
		FechaCreacion: now,
	}, nil
}

func VerificarPassword(u models.Usuario, password string) bool {
	esperado := derivarClave(password, u.Salt)
	return subtle.ConstantTimeCompare([]byte(esperado), []byte(u.HashPassword)) == 1
}

func BuscarUsuario(usuarios []models.Usuario, identificador string) (models.Usuario, int, bool) {
	identificador = strings.TrimSpace(identificador)
	for i, u := range usuarios {
		if strings.EqualFold(u.Usuario, identificador) || strings.EqualFold(u.Email, identificador) {
			return u, i, true
		}
	}
	return models.Usuario{}, -1, false
}

func salAleatoria() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// derivarClave implementa PBKDF2-SHA256 con la biblioteca estándar, sin
// dependencias externas.
func derivarClave(password, salt string) string {
	return hex.EncodeToString(pbkdf2([]byte(password), []byte(salt), iteracionesPBKDF2, largoClaveHash))
}

func pbkdf2(password, salt []byte, iteraciones, largo int) []byte {
	hLen := sha256.Size
	bloques := (largo + hLen - 1) / hLen
	salida := make([]byte, 0, bloques*hLen)

	for bloque := 1; bloque <= bloques; bloque++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(bloque >> 24), byte(bloque >> 16), byte(bloque >> 8), byte(bloque)})
		u := mac.Sum(nil)
		acumulado := make([]byte, len(u))
		copy(acumulado, u)

		for i := 1; i < iteraciones; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range acumulado {
				acumulado[j] ^= u[j]
			}
		}
		salida = append(salida, acumulado...)
	}
	return salida[:largo]
}

// TokenSesion genera un identificador de sesión imposible de adivinar.
func TokenSesion() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

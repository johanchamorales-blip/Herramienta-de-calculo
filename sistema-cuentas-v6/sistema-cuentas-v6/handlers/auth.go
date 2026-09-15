package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

const (
	cookieSesion    = "sesion_cuentas"
	duracionSesion  = 8 * time.Hour
	claveSesionCtx  = "sesion"
)

type sesion struct {
	Usuario models.Usuario
	Expira  time.Time
}

var (
	sesionesMu sync.RWMutex
	sesiones   = map[string]sesion{}
)

// SesionDe devuelve el usuario autenticado de la petición.
func SesionDe(r *http.Request) (models.Usuario, bool) {
	cookie, err := r.Cookie(cookieSesion)
	if err != nil {
		return models.Usuario{}, false
	}
	sesionesMu.RLock()
	s, ok := sesiones[cookie.Value]
	sesionesMu.RUnlock()
	if !ok {
		return models.Usuario{}, false
	}
	if time.Now().After(s.Expira) {
		cerrarSesion(cookie.Value)
		return models.Usuario{}, false
	}
	return s.Usuario, true
}

func abrirSesion(w http.ResponseWriter, u models.Usuario) error {
	token, err := services.TokenSesion()
	if err != nil {
		return err
	}
	sesionesMu.Lock()
	sesiones[token] = sesion{Usuario: u, Expira: time.Now().Add(duracionSesion)}
	sesionesMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieSesion,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(duracionSesion.Seconds()),
	})
	return nil
}

func cerrarSesion(token string) {
	sesionesMu.Lock()
	delete(sesiones, token)
	sesionesMu.Unlock()
}

// RequiereSesion protege un endpoint. Si se indica un permiso, además se exige
// que el rol del usuario lo tenga.
func RequiereSesion(permiso string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usuario, ok := SesionDe(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "inicia sesión para continuar")
			return
		}
		// Las lecturas están abiertas a cualquier rol autenticado; los permisos
		// solo se exigen a las operaciones que modifican datos.
		if permiso != "" && r.Method != http.MethodGet && !services.Puede(usuario.Rol, permiso) {
			writeError(w, http.StatusForbidden, "tu rol de "+services.NombreRol(usuario.Rol)+" no puede realizar esta acción")
			return
		}
		next(w, r)
	}
}

// Registro crea usuarios. El primero se vuelve administrador; a partir de ahí
// solo un administrador puede dar de alta a otros.
func Registro(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Usuario  string `json:"usuario"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Rol      string `json:"rol"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}

	err := enTransaccion(func() error {
		usuarios, err := usuarioStore.Read()
		if err != nil {
			return err
		}
		if len(usuarios) > 0 {
			actual, ok := SesionDe(r)
			if !ok || !services.Puede(actual.Rol, "usuarios") {
				writeError(w, http.StatusForbidden, "solo un administrador puede crear usuarios")
				return nil
			}
		}
		nuevo, err := services.CrearUsuario(usuarios, input.Usuario, input.Email, input.Password, input.Rol, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}
		usuarios = append(usuarios, nuevo)
		if err := usuarioStore.Write(usuarios); err != nil {
			return err
		}
		auditar(r, "CREÓ USUARIO", nuevo.Usuario, "Rol "+services.NombreRol(nuevo.Rol))
		writeJSON(w, http.StatusCreated, nuevo.Publico())
		return nil
	})
	if err != nil {
		responderError(w, err)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Usuario  string `json:"usuario"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}

	usuarios, err := usuarioStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	usuario, idx, ok := services.BuscarUsuario(usuarios, input.Usuario)
	// Se responde igual con usuario inexistente o contraseña incorrecta para no
	// revelar qué usuarios existen.
	if !ok || !usuario.Activo || !services.VerificarPassword(usuario, input.Password) {
		writeError(w, http.StatusUnauthorized, "usuario o contraseña incorrectos")
		return
	}

	usuarios[idx].UltimoAcceso = time.Now().UTC()
	_ = usuarioStore.Write(usuarios)

	if err := abrirSesion(w, usuario); err != nil {
		responderError(w, err)
		return
	}
	auditar(r, "INICIÓ SESIÓN", usuario.Usuario, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"usuario":  usuario.Publico(),
		"permisos": services.Permisos(usuario.Rol),
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieSesion); err == nil {
		cerrarSesion(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieSesion, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	writeJSON(w, http.StatusOK, map[string]string{"mensaje": "sesión cerrada"})
}

// Sesion informa al navegador quién está conectado y si hay que mostrar el
// alta del primer administrador.
func Sesion(w http.ResponseWriter, r *http.Request) {
	usuarios, err := usuarioStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	respuesta := map[string]any{
		"autenticado":     false,
		"requiere_alta":   len(usuarios) == 0,
		"roles":           services.RolesDisponibles(),
	}
	if usuario, ok := SesionDe(r); ok {
		respuesta["autenticado"] = true
		respuesta["usuario"] = usuario.Publico()
		respuesta["permisos"] = services.Permisos(usuario.Rol)
	}
	writeJSON(w, http.StatusOK, respuesta)
}

// Usuarios lista las cuentas de acceso (sin credenciales) y permite activarlas
// o desactivarlas.
func Usuarios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		usuarios, err := usuarioStore.Read()
		if err != nil {
			responderError(w, err)
			return
		}
		salida := make([]map[string]any, 0, len(usuarios))
		for _, u := range usuarios {
			publico := u.Publico()
			publico["ultimo_acceso"] = u.UltimoAcceso
			salida = append(salida, publico)
		}
		writeJSON(w, http.StatusOK, salida)
	case http.MethodPatch:
		var input struct {
			ID     int    `json:"id"`
			Activo *bool  `json:"activo"`
			Rol    string `json:"rol"`
		}
		if err := readJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "revisa los datos enviados")
			return
		}
		actual, _ := SesionDe(r)
		if actual.ID == input.ID {
			writeError(w, http.StatusBadRequest, "no puedes modificar tu propio acceso")
			return
		}
		err := enTransaccion(func() error {
			usuarios, err := usuarioStore.Read()
			if err != nil {
				return err
			}
			for i := range usuarios {
				if usuarios[i].ID != input.ID {
					continue
				}
				if input.Activo != nil {
					usuarios[i].Activo = *input.Activo
				}
				if rol := strings.ToUpper(strings.TrimSpace(input.Rol)); rol != "" {
					if len(services.Permisos(rol)) == 0 && rol != models.RolJefePlanta {
						writeError(w, http.StatusBadRequest, "rol inválido")
						return nil
					}
					usuarios[i].Rol = rol
				}
				if err := usuarioStore.Write(usuarios); err != nil {
					return err
				}
				auditar(r, "ACTUALIZÓ USUARIO", usuarios[i].Usuario, "Rol "+services.NombreRol(usuarios[i].Rol))
				writeJSON(w, http.StatusOK, usuarios[i].Publico())
				return nil
			}
			writeError(w, http.StatusNotFound, "usuario no encontrado")
			return nil
		})
		if err != nil {
			responderError(w, err)
		}
	default:
		methodNotAllowed(w)
	}
}

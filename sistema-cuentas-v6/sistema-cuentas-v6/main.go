package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sistema-cuentas/handlers"
)

func main() {
	if err := prepararDatos(); err != nil { log.Fatal(err) }
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.Home)
	mux.HandleFunc("/api/auth/sesion", handlers.Sesion)
	mux.HandleFunc("/api/auth/login", handlers.Login)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)
	mux.HandleFunc("/api/auth/registro", handlers.Registro)
	mux.HandleFunc("/api/cooperativas", handlers.RequiereSesion("catalogos", handlers.Cooperativas))
	mux.HandleFunc("/api/bancos", handlers.RequiereSesion("catalogos", handlers.Bancos))
	mux.HandleFunc("/api/cuentas", handlers.RequiereSesion("catalogos", handlers.Cuentas))
	mux.HandleFunc("/api/movimientos", handlers.RequiereSesion("movimientos", handlers.Movimientos))
	mux.HandleFunc("/api/movimientos/cobrar", handlers.RequiereSesion("movimientos", handlers.CobrarCheque))
	mux.HandleFunc("/api/libro-bancos", handlers.RequiereSesion("", handlers.LibroBancos))
	mux.HandleFunc("/api/libro-bancos.csv", handlers.RequiereSesion("", handlers.LibroBancosCSV))
	mux.HandleFunc("/api/conciliaciones", handlers.RequiereSesion("conciliacion", handlers.Conciliaciones))
	mux.HandleFunc("/api/conciliacion", handlers.RequiereSesion("", handlers.ConciliacionPreview))
	mux.HandleFunc("/api/conciliacion/detalle", handlers.RequiereSesion("", handlers.ConciliacionDetalle))
	mux.HandleFunc("/api/conciliacion/importar", handlers.RequiereSesion("conciliacion", handlers.ImportarEstadoCuenta))
	mux.HandleFunc("/api/busqueda", handlers.RequiereSesion("", handlers.Busqueda))
	mux.HandleFunc("/api/auditoria", handlers.RequiereSesion("", handlers.Auditoria))
	mux.HandleFunc("/api/resumen", handlers.RequiereSesion("", handlers.Resumen))
	mux.HandleFunc("/api/usuarios", handlers.RequiereSesion("usuarios", handlers.Usuarios))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	addr := os.Getenv("PORT"); if addr == "" { addr = ":8080" }
	servidor := &http.Server{Addr:addr, Handler:registrarPeticiones(mux), ReadTimeout:15*time.Second, WriteTimeout:30*time.Second, IdleTimeout:60*time.Second}
	fmt.Printf("Sistema de movimientos de cuentas en http://localhost%s\n", addr)
	log.Fatal(servidor.ListenAndServe())
}

func prepararDatos() error {
	if err := os.MkdirAll("data", 0o755); err != nil { return err }
	archivos := []string{"cooperativas.json","bancos.json","cuentas.json","movimientos.json","conciliaciones.json","auditoria.json","usuarios.json"}
	for _, nombre := range archivos { ruta:=filepath.Join("data",nombre); if _,err:=os.Stat(ruta); os.IsNotExist(err) { if err:=os.WriteFile(ruta,[]byte("[]\n"),0o644); err!=nil{return err} } }
	return nil
}
func registrarPeticiones(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ inicio:=time.Now(); next.ServeHTTP(w,r); if r.URL.Path!="/"&&!filepath.HasPrefix(r.URL.Path,"/static"){log.Printf("%s %s (%s)",r.Method,r.URL.Path,time.Since(inicio).Round(time.Millisecond))} }) }

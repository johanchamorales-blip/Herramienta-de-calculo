# Sistema de movimientos de cuentas — Libro de Bancos

Aplicación web en Go (sin dependencias externas) para llevar el libro de bancos
de varias cooperativas: catálogos, ingresos y egresos, saldo corrido,
conciliación bancaria y bitácora de auditoría. Los datos se guardan en archivos
JSON dentro de `data/`.

## Ejecutar

```bash
go test ./...
go run .
```

Luego abrir <http://localhost:8080>. El primer usuario que se registre queda
como **Administrador**; a partir de ahí solo un administrador puede crear
accesos nuevos.

## Roles

| Rol | Puede |
| --- | --- |
| Administrador | Todo, incluida la gestión de accesos |
| Contador | Catálogos, movimientos y conciliación |
| Jefe de planta | Consultar libros, reportes y bitácora |

Las lecturas están abiertas a cualquier sesión válida; los permisos se exigen
en las operaciones que modifican datos.

## Organización del código

```
models/      Modelos y el tipo Fecha (día calendario, sin zona horaria)
services/    Reglas de negocio: dinero, saldos, movimientos, libro,
             conciliación, búsqueda difusa, NIT, usuarios
handlers/    Endpoints HTTP, sesiones y transacción global
static/css/  tokens · base · app · print
static/js/core/   api · estado · vista · pestanas · dom · formato ·
                  notificaciones · dialogo
static/js/views/  una clase por pestaña
```

Cada pantalla es una clase que hereda de `Vista` (`plantilla`, `conectar`,
`actualizar`, `destruir`). Para agregar un módulo nuevo basta con crear el
archivo en `views/` y registrarlo en `static/js/app.js`.

## Decisiones de precisión

- **Centavos.** Todo importe pasa por `services.Q`, que redondea a dos
  decimales. Evita que el saldo corrido se desvíe tras cientos de movimientos.
- **Fechas por día.** `models.Fecha` se serializa como `AAAA-MM-DD`. Un cheque
  del 30 de septiembre a las 20:00 en Guatemala ya no se contabiliza en octubre
  al convertirse a UTC. Al leer, acepta también el formato RFC3339 de versiones
  anteriores, así que los archivos JSON existentes siguen funcionando.
- **El saldo se deriva.** `services.SaldoDeCuenta` lo recalcula desde el saldo
  inicial y los movimientos vigentes; el campo `saldo_actual` es solo caché.
- **Una escritura a la vez.** `handlers.enTransaccion` serializa las
  operaciones que tocan varios archivos, para que dos registros simultáneos no
  se pisen ni repitan un ID.
- **El sobregiro avisa, no bloquea.** Es un hecho contable posible; el libro
  debe reflejar lo que pasó en el banco.
- **NIT verificado.** Se valida el dígito verificador módulo 11 (se aceptan los
  NIT de 13 dígitos derivados del CUI).

## Conciliación bancaria

Se calculan las dos mitades por separado y la diferencia sale sola:

```
Banco:  saldo del estado de cuenta + depósitos en tránsito − cheques en circulación
Libros: saldo en libros + notas de crédito − notas de débito ± ajustes
```

Los cheques en circulación se detectan solos: son los egresos que siguen en
estado `EMITIDO` al cierre del periodo, más los que el banco pagó después del
cierre. Por eso es importante marcar el cobro de cada cheque desde la pestaña
de Egresos.

## Endpoints

| Método | Ruta | Uso |
| --- | --- | --- |
| GET/POST | `/api/auth/sesion`, `/api/auth/login`, `/api/auth/logout`, `/api/auth/registro` | Acceso |
| GET/POST/PUT/DELETE | `/api/cooperativas`, `/api/bancos`, `/api/cuentas` | Catálogos |
| GET/POST/DELETE | `/api/movimientos` | Ingresos y egresos (DELETE = anulación lógica, exige `motivo`) |
| POST | `/api/movimientos/cobrar` | Registrar que el banco pagó un cheque |
| GET | `/api/libro-bancos`, `/api/libro-bancos.csv` | Libro con saldo corrido y exportación |
| GET/POST | `/api/conciliacion`, `/api/conciliaciones` | Calcular y guardar conciliaciones |
| GET | `/api/resumen`, `/api/busqueda`, `/api/auditoria` | Panel, búsqueda difusa y bitácora |
| GET/PATCH | `/api/usuarios` | Accesos y roles |

## Impresión

El libro de bancos y la conciliación se imprimen (Ctrl/Cmd + P) como documento
formal: encabezado con cooperativa, NIT, dirección, cuenta y banco, y pie con
las firmas de Elaboró, Tesorero, Vo. Bo. y Presidente de la Comisión de
Vigilancia.

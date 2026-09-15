import { api, alPerderSesion } from './core/api.js';
import { estado } from './core/estado.js';
import { Escritorio } from './core/pestanas.js';
import { problema } from './core/notificaciones.js';
import { Acceso } from './views/acceso.js';
import { Inicio } from './views/inicio.js';
import { Cooperativas } from './views/cooperativas.js';
import { Bancos } from './views/bancos.js';
import { Cuentas } from './views/cuentas.js';
import { Movimientos, Ingresos, Egresos } from './views/movimientos.js';
import { LibroBancos } from './views/libro.js';
import { Conciliaciones } from './views/conciliaciones.js';
import { Busqueda } from './views/busqueda.js';
import { Auditoria } from './views/auditoria.js';
import { Usuarios } from './views/usuarios.js';

const raiz = document.getElementById('raiz');

// Un módulo por pestaña. Agregar una pantalla nueva es agregar una clase aquí.
const REGISTRO = {
  inicio: Inicio,
  cooperativas: Cooperativas,
  bancos: Bancos,
  cuentas: Cuentas,
  movimientos: Movimientos,
  ingresos: Ingresos,
  egresos: Egresos,
  libro: LibroBancos,
  conciliaciones: Conciliaciones,
  busqueda: Busqueda,
  auditoria: Auditoria,
  usuarios: Usuarios,
};

const GRUPOS = [
  { titulo: 'Inicio', modulos: ['inicio', 'busqueda'] },
  { titulo: 'Catálogos', modulos: ['cooperativas', 'bancos', 'cuentas'] },
  { titulo: 'Operación', modulos: ['movimientos', 'ingresos', 'egresos'] },
  { titulo: 'Reportes', modulos: ['libro', 'conciliaciones'] },
  { titulo: 'Administración', modulos: ['auditoria', 'usuarios'] },
];

const acceso = new Acceso(raiz, iniciarSesionCompleta);

async function arrancar() {
  try {
    const sesion = await api.obtener('/api/auth/sesion');
    if (sesion.autenticado) {
      await iniciarSesionCompleta(sesion);
    } else {
      await acceso.mostrar(sesion);
    }
  } catch (error) {
    problema(error.message);
  }
}

async function iniciarSesionCompleta(sesion) {
  estado.usuario = sesion.usuario;
  estado.permisos = sesion.permisos || [];
  await estado.cargarCatalogos();

  // El menú se arma según el rol: quien no administra accesos no ve el módulo.
  const grupos = GRUPOS.map((grupo) => ({
    ...grupo,
    modulos: grupo.modulos.filter((id) => id !== 'usuarios' || estado.puede('usuarios')),
  })).filter((grupo) => grupo.modulos.length);

  const escritorio = new Escritorio({ contenedor: raiz, grupos, registro: REGISTRO });
  await escritorio.abrir('inicio');

  raiz.querySelector('[data-accion="salir"]').addEventListener('click', async () => {
    await api.crear('/api/auth/logout', {});
    window.location.reload();
  });
}

alPerderSesion(() => {
  setTimeout(() => window.location.reload(), 1200);
});

arrancar();

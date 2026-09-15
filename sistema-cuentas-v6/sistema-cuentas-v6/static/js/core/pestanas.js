import { estado } from './estado.js';
import { escapar, opciones } from './dom.js';
import { dinero } from './formato.js';
import { problema } from './notificaciones.js';

// Gestiona el escritorio de trabajo: menú lateral, barra de pestañas y la
// pestaña activa. Cada botón del menú abre su propia pestaña, y las pestañas
// abiertas conservan lo que el usuario tenía en pantalla.
export class Escritorio {
  constructor({ contenedor, grupos, registro }) {
    this.contenedor = contenedor;
    this.grupos = grupos;       // [{ titulo, modulos: ['inicio', ...] }]
    this.registro = registro;   // { inicio: ClaseVista, ... }
    this.abiertas = new Map();  // id -> { clase, instancia, obsoleta }
    this.activa = null;

    this.contenedor.innerHTML = this.plantilla();
    this.menu = this.contenedor.querySelector('.menu');
    this.barra = this.contenedor.querySelector('.barra-pestanas');
    this.lienzo = this.contenedor.querySelector('.lienzo');
    this.contexto = this.contenedor.querySelector('.contexto');

    this.conectar();
    this.pintarMenu();
    this.pintarContexto();
  }

  plantilla() {
    return `
      <div class="aplicacion">
        <aside class="lateral">
          <div class="marca">
            <strong>Libro de Bancos</strong>
            <span>Movimientos de cuentas</span>
          </div>
          <nav class="menu" aria-label="Módulos del sistema"></nav>
          <div class="sesion">
            <div>
              <div class="quien"></div>
              <div class="rol"></div>
            </div>
            <button type="button" class="secundario" data-accion="salir">Cerrar sesión</button>
          </div>
        </aside>
        <main class="trabajo">
          <div class="barra-pestanas" role="tablist"></div>
          <div class="contexto"></div>
          <div class="lienzo"></div>
        </main>
      </div>`;
  }

  conectar() {
    this.menu.addEventListener('click', (evento) => {
      const boton = evento.target.closest('button[data-modulo]');
      if (boton) this.abrir(boton.dataset.modulo);
    });

    this.barra.addEventListener('click', (evento) => {
      const cerrar = evento.target.closest('button[data-cerrar]');
      if (cerrar) {
        evento.stopPropagation();
        this.cerrar(cerrar.dataset.cerrar);
        return;
      }
      const pestana = evento.target.closest('button[data-pestana]');
      if (pestana) this.activar(pestana.dataset.pestana);
    });

    this.contexto.addEventListener('change', (evento) => {
      if (evento.target.id === 'contexto-cooperativa') estado.elegirCooperativa(evento.target.value);
      if (evento.target.id === 'contexto-cuenta') estado.elegirCuenta(evento.target.value);
    });

    estado.addEventListener('cambio', () => {
      this.pintarContexto();
      this.refrescarActiva();
    });
  }

  pintarMenu() {
    this.menu.innerHTML = this.grupos.map((grupo) => `
      <div class="menu-grupo">
        <h4>${escapar(grupo.titulo)}</h4>
        ${grupo.modulos.map((id) => {
          const clase = this.registro[id];
          if (!clase) return '';
          return `<button type="button" data-modulo="${id}" aria-current="false">
            <span class="glifo" aria-hidden="true">${clase.glifo}</span>${escapar(clase.titulo)}
          </button>`;
        }).join('')}
      </div>`).join('');
  }

  pintarSesion() {
    const quien = this.contenedor.querySelector('.sesion .quien');
    const rol = this.contenedor.querySelector('.sesion .rol');
    if (estado.usuario) {
      quien.textContent = estado.usuario.usuario;
      rol.textContent = estado.usuario.rol.replaceAll('_', ' ').toLowerCase();
    }
  }

  pintarContexto() {
    const cuentas = estado.cuentasDeCooperativa();
    const cuenta = estado.cuenta;
    this.contexto.innerHTML = `
      <label for="contexto-cooperativa">Cooperativa
        <select id="contexto-cooperativa">
          ${opciones(estado.cooperativas, {
            valor: (c) => c.id,
            texto: (c) => c.nombre,
            seleccionado: estado.cooperativaID,
            vacio: 'Todas',
          })}
        </select>
      </label>
      <label for="contexto-cuenta">Cuenta
        <select id="contexto-cuenta">
          ${opciones(cuentas, {
            valor: (c) => c.id,
            texto: (c) => estado.etiquetaCuenta(c),
            seleccionado: estado.cuentaID,
            vacio: cuentas.length ? 'Sin seleccionar' : 'No hay cuentas registradas',
          })}
        </select>
      </label>
      <div class="saldo-contexto">
        <span class="tenue">Saldo</span>
        <strong class="cifra ${cuenta && cuenta.saldo_actual < 0 ? 'negativo' : ''}">${cuenta ? dinero(cuenta.saldo_actual) : '—'}</strong>
      </div>`;
    this.pintarSesion();
  }

  pintarBarra() {
    this.barra.innerHTML = Array.from(this.abiertas.entries()).map(([id, pestana]) => `
      <button type="button" class="pestana" role="tab" data-pestana="${id}"
              aria-selected="${id === this.activa}">
        <span class="glifo" aria-hidden="true">${pestana.clase.glifo}</span>
        ${escapar(pestana.clase.titulo)}
        ${id === 'inicio' ? '' : `<span class="cerrar" role="button" tabindex="-1" data-cerrar="${id}" title="Cerrar pestaña">×</span>`}
      </button>`).join('');
  }

  async abrir(id) {
    const clase = this.registro[id];
    if (!clase) return;

    if (!this.abiertas.has(id)) {
      const instancia = new clase({
        estado,
        abrirPestana: (otro) => this.abrir(otro),
        refrescarCatalogos: () => estado.cargarCatalogos(),
      });
      this.abiertas.set(id, { clase, instancia, montada: false });
      this.lienzo.appendChild(instancia.nodo);
    }
    await this.activar(id);
  }

  async activar(id) {
    const pestana = this.abiertas.get(id);
    if (!pestana) return;

    this.activa = id;
    this.abiertas.forEach((otra, otroID) => {
      otra.instancia.nodo.style.display = otroID === id ? '' : 'none';
    });
    this.pintarBarra();
    this.menu.querySelectorAll('button[data-modulo]').forEach((boton) => {
      boton.setAttribute('aria-current', String(boton.dataset.modulo === id));
    });

    try {
      if (!pestana.montada) {
        await pestana.instancia.montar();
        pestana.montada = true;
      } else if (pestana.obsoleta) {
        await pestana.instancia.actualizar();
      }
      pestana.obsoleta = false;
    } catch (error) {
      problema(error.message);
    }
    this.lienzo.scrollTop = 0;
  }

  cerrar(id) {
    const pestana = this.abiertas.get(id);
    if (!pestana || id === 'inicio') return;
    pestana.instancia.destruir();
    pestana.instancia.nodo.remove();
    this.abiertas.delete(id);
    if (this.activa === id) {
      const siguiente = Array.from(this.abiertas.keys()).pop() || 'inicio';
      this.abrir(siguiente);
    } else {
      this.pintarBarra();
    }
  }

  // Al cambiar de cuenta solo se recarga la pestaña visible; las demás se
  // marcan para recargarse cuando el usuario vuelva a ellas.
  async refrescarActiva() {
    for (const [id, pestana] of this.abiertas) {
      if (id === this.activa && pestana.montada) {
        try {
          await pestana.instancia.actualizar();
        } catch (error) {
          problema(error.message);
        }
      } else {
        pestana.obsoleta = true;
      }
    }
  }
}

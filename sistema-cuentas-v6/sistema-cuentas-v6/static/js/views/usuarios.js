import { Vista } from '../core/vista.js';
import { api } from '../core/api.js';
import { escapar, datosFormulario } from '../core/dom.js';
import { fechaHora, nombreRol } from '../core/formato.js';
import { exito, problema } from '../core/notificaciones.js';

// Administración de accesos. Solo la ve el rol Administrador.
export class Usuarios extends Vista {
  static titulo = 'Usuarios y roles';
  static glifo = '☖';

  plantilla() {
    return `
      ${this.encabezado('Usuarios y roles', 'El administrador gestiona catálogos y accesos; el contador registra movimientos y concilia; el jefe de planta consulta.')}
      <div class="columnas">
        <div class="tarjeta">
          <h2>Nuevo acceso</h2>
          <form class="formulario" data-formulario="usuario" style="margin-top:14px">
            <label>Usuario<input name="usuario" required minlength="3" autocomplete="off"></label>
            <label>Correo<input name="email" type="email" required autocomplete="off"></label>
            <label>Contraseña<input name="password" type="password" required minlength="8" autocomplete="new-password"></label>
            <label>Rol
              <select name="rol" required>
                <option value="CONTADOR">Contador</option>
                <option value="JEFE_DE_PLANTA">Jefe de planta</option>
                <option value="ADMINISTRADOR">Administrador</option>
              </select>
            </label>
            <div class="acciones"><button type="submit">Crear acceso</button></div>
          </form>
        </div>
        <div class="tarjeta">
          <h2>Accesos existentes</h2>
          <div class="lista" data-zona="lista" style="margin-top:14px"></div>
        </div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="usuario"]', (form) => this.crear(form));
    this.alHacerClic('[data-activar]', (boton) => this.cambiarEstado(Number(boton.dataset.activar), boton.dataset.valor === 'true'));
  }

  async crear(form) {
    try {
      await api.crear('/api/auth/registro', datosFormulario(form));
      form.reset();
      exito('Acceso creado');
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async cambiarEstado(id, activo) {
    try {
      await api.parchear('/api/usuarios', { id, activo });
      exito(activo ? 'Acceso habilitado' : 'Acceso deshabilitado');
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async actualizar() {
    const usuarios = await api.obtener('/api/usuarios');
    this.$('[data-zona="lista"]').innerHTML = usuarios.map((u) => `
      <div class="elemento">
        <div>
          <strong>${escapar(u.usuario)}</strong>
          <small>${escapar(u.email)} · ${escapar(nombreRol(u.rol))}</small>
          <small>${u.ultimo_acceso && !u.ultimo_acceso.startsWith('0001') ? `Último acceso: ${fechaHora(u.ultimo_acceso)}` : 'Sin accesos todavía'}</small>
        </div>
        <div class="controles">
          <span class="etiqueta ${u.activo ? 'activo' : 'anulado'}">${u.activo ? 'Activo' : 'Inactivo'}</span>
          <button type="button" class="enlace" data-activar="${u.id}" data-valor="${!u.activo}">
            ${u.activo ? 'Deshabilitar' : 'Habilitar'}
          </button>
        </div>
      </div>`).join('');
  }
}

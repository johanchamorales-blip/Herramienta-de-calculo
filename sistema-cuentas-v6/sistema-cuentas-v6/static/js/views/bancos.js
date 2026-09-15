import { Vista } from '../core/vista.js';
import { api } from '../core/api.js';
import { escapar, datosFormulario } from '../core/dom.js';
import { dinero } from '../core/formato.js';
import { exito, problema } from '../core/notificaciones.js';
import { confirmar } from '../core/dialogo.js';

export class Bancos extends Vista {
  static titulo = 'Bancos';
  static glifo = '▤';

  plantilla() {
    return `
      ${this.encabezado('Bancos', 'Cada banco muestra cuántas cooperativas y cuentas tiene asociadas.')}
      <div class="columnas">
        <div class="tarjeta">
          <h2>Nuevo banco</h2>
          <form class="formulario" data-formulario="banco" style="margin-top:14px">
            <input type="hidden" name="id">
            <label>Nombre<input name="nombre" required placeholder="Ej. Banco Industrial"></label>
            <div class="acciones">
              <button type="submit">Guardar banco</button>
              <button type="button" class="secundario oculto" data-accion="cancelar">Cancelar edición</button>
            </div>
          </form>
        </div>
        <div class="tarjeta">
          <h2>Registrados</h2>
          <div class="lista" data-zona="lista" style="margin-top:14px"></div>
        </div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="banco"]', (form) => this.guardar(form));
    this.alHacerClic('[data-accion="cancelar"]', () => this.limpiar());
    this.alHacerClic('[data-editar]', (boton) => {
      const banco = this.estado.bancos.find((b) => b.id === Number(boton.dataset.editar));
      if (!banco) return;
      const form = this.$('[data-formulario="banco"]');
      form.id.value = banco.id;
      form.nombre.value = banco.nombre;
      this.$('[data-accion="cancelar"]').classList.remove('oculto');
      form.nombre.focus();
    });
    this.alHacerClic('[data-eliminar]', (boton) => this.eliminar(Number(boton.dataset.eliminar)));
  }

  limpiar() {
    const form = this.$('[data-formulario="banco"]');
    form.reset();
    form.id.value = '';
    this.$('[data-accion="cancelar"]').classList.add('oculto');
  }

  async guardar(form) {
    const datos = datosFormulario(form);
    try {
      if (datos.id) {
        await api.actualizar('/api/bancos', { id: Number(datos.id), nombre: datos.nombre });
        exito('Banco actualizado');
      } else {
        await api.crear('/api/bancos', { nombre: datos.nombre });
        exito('Banco registrado');
      }
      this.limpiar();
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async eliminar(id) {
    const respuesta = await confirmar('Eliminar banco', 'Solo puede eliminarse si no tiene cuentas asociadas.', {
      confirmar: 'Eliminar', peligro: true,
    });
    if (!respuesta) return;
    try {
      await api.eliminar(`/api/bancos?id=${id}`);
      exito('Banco eliminado');
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async actualizar() {
    const lista = this.$('[data-zona="lista"]');
    if (!this.estado.bancos.length) {
      lista.innerHTML = '<div class="vacio">Agrega los bancos con los que trabajan las cooperativas.</div>';
      return;
    }
    lista.innerHTML = this.estado.bancos.map((banco) => {
      const cuentas = this.estado.cuentas.filter((c) => c.banco_id === banco.id);
      const cooperativas = new Set(cuentas.map((c) => c.cooperativa_id)).size;
      const saldo = cuentas.reduce((total, c) => total + (c.saldo_actual || 0), 0);
      return `
        <div class="elemento">
          <div>
            <strong>${escapar(banco.nombre)}</strong>
            <small>${cuentas.length} cuenta(s) · ${cooperativas} cooperativa(s) asociada(s)</small>
            <small class="cifra">Saldo agrupado: ${dinero(saldo)}</small>
          </div>
          <div class="controles">
            <button type="button" class="enlace" data-editar="${banco.id}">Editar</button>
            <button type="button" class="enlace" data-eliminar="${banco.id}">Eliminar</button>
          </div>
        </div>`;
    }).join('');
  }
}

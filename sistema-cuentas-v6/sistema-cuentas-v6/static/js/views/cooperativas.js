import { Vista } from '../core/vista.js';
import { api } from '../core/api.js';
import { escapar, datosFormulario } from '../core/dom.js';
import { exito, problema } from '../core/notificaciones.js';
import { confirmar } from '../core/dialogo.js';

export class Cooperativas extends Vista {
  static titulo = 'Cooperativas';
  static glifo = '⬡';

  plantilla() {
    return `
      ${this.encabezado('Cooperativas', 'El NIT identifica a la cooperativa y se verifica con su dígito verificador.')}
      <div class="columnas">
        <div class="tarjeta">
          <h2 data-zona="titulo-formulario">Nueva cooperativa</h2>
          <form class="formulario" data-formulario="cooperativa" style="margin-top:14px">
            <input type="hidden" name="id">
            <label>Nombre<input name="nombre" required placeholder="Ej. Micoope"></label>
            <label>NIT<input name="nit" required placeholder="1234567-8" autocomplete="off"></label>
            <label>Dirección<input name="direccion" placeholder="Zona, municipio, departamento"></label>
            <label>Teléfono<input name="telefono" placeholder="Opcional"></label>
            <div class="acciones">
              <button type="submit">Guardar cooperativa</button>
              <button type="button" class="secundario oculto" data-accion="cancelar">Cancelar edición</button>
            </div>
          </form>
        </div>
        <div class="tarjeta">
          <h2>Registradas</h2>
          <div class="lista" data-zona="lista" style="margin-top:14px"></div>
        </div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="cooperativa"]', (form) => this.guardar(form));
    this.alHacerClic('[data-accion="cancelar"]', () => this.limpiar());
    this.alHacerClic('[data-editar]', (boton) => this.editar(Number(boton.dataset.editar)));
    this.alHacerClic('[data-eliminar]', (boton) => this.eliminar(Number(boton.dataset.eliminar)));
    this.alHacerClic('[data-cuentas]', () => this.ctx.abrirPestana('cuentas'));
  }

  async guardar(form) {
    const datos = datosFormulario(form);
    const carga = {
      nombre: datos.nombre,
      nit: datos.nit,
      direccion: datos.direccion || '',
      telefono: datos.telefono || '',
    };
    try {
      if (datos.id) {
        carga.id = Number(datos.id);
        await api.actualizar('/api/cooperativas', carga);
        exito('Cooperativa actualizada');
      } else {
        await api.crear('/api/cooperativas', carga);
        exito('Cooperativa registrada');
      }
      this.limpiar();
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  editar(id) {
    const cooperativa = this.estado.cooperativas.find((c) => c.id === id);
    if (!cooperativa) return;
    const form = this.$('[data-formulario="cooperativa"]');
    form.id.value = cooperativa.id;
    form.nombre.value = cooperativa.nombre;
    form.nit.value = cooperativa.nit;
    form.direccion.value = cooperativa.direccion || '';
    form.telefono.value = cooperativa.telefono || '';
    this.$('[data-zona="titulo-formulario"]').textContent = `Editando ${cooperativa.nombre}`;
    this.$('[data-accion="cancelar"]').classList.remove('oculto');
    form.nombre.focus();
  }

  limpiar() {
    const form = this.$('[data-formulario="cooperativa"]');
    form.reset();
    form.id.value = '';
    this.$('[data-zona="titulo-formulario"]').textContent = 'Nueva cooperativa';
    this.$('[data-accion="cancelar"]').classList.add('oculto');
  }

  async eliminar(id) {
    const cooperativa = this.estado.cooperativas.find((c) => c.id === id);
    const respuesta = await confirmar(
      'Eliminar cooperativa',
      `Se quitará ${cooperativa?.nombre || 'la cooperativa'} del catálogo. Solo es posible si no tiene cuentas bancarias.`,
      { confirmar: 'Eliminar', peligro: true },
    );
    if (!respuesta) return;
    try {
      await api.eliminar(`/api/cooperativas?id=${id}`);
      exito('Cooperativa eliminada');
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async actualizar() {
    const lista = this.$('[data-zona="lista"]');
    const cooperativas = this.estado.cooperativas;
    if (!cooperativas.length) {
      lista.innerHTML = '<div class="vacio">Registra la primera cooperativa con su NIT para habilitar cuentas y movimientos.</div>';
      return;
    }
    lista.innerHTML = cooperativas.map((c) => {
      const cuentas = this.estado.cuentas.filter((x) => x.cooperativa_id === c.id).length;
      return `
        <div class="elemento">
          <div>
            <strong>${escapar(c.nombre)}</strong>
            <small>NIT ${escapar(c.nit)}${c.direccion ? ` · ${escapar(c.direccion)}` : ''}${c.telefono ? ` · ${escapar(c.telefono)}` : ''}</small>
            <small>${cuentas} cuenta(s) bancaria(s)</small>
          </div>
          <div class="controles">
            <button type="button" class="enlace" data-cuentas>Ver cuentas</button>
            <button type="button" class="enlace" data-editar="${c.id}">Editar</button>
            <button type="button" class="enlace" data-eliminar="${c.id}">Eliminar</button>
          </div>
        </div>`;
    }).join('');
  }
}

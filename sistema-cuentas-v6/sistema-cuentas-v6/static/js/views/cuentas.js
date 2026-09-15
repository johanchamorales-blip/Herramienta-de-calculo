import { Vista } from '../core/vista.js';
import { api } from '../core/api.js';
import { escapar, datosFormulario, opciones } from '../core/dom.js';
import { dinero } from '../core/formato.js';
import { exito, problema } from '../core/notificaciones.js';
import { confirmar } from '../core/dialogo.js';

export class Cuentas extends Vista {
  static titulo = 'Cuentas bancarias';
  static glifo = '▣';

  plantilla() {
    return `
      ${this.encabezado('Cuentas bancarias', 'Cada cuenta pertenece a una cooperativa y a un banco. El saldo inicial solo puede fijarse antes del primer movimiento.')}
      <div class="columnas">
        <div class="tarjeta">
          <h2 data-zona="titulo-formulario">Nueva cuenta</h2>
          <form class="formulario" data-formulario="cuenta" style="margin-top:14px">
            <input type="hidden" name="id">
            <label>Cooperativa<select name="cooperativa_id" required data-zona="cooperativas"></select></label>
            <label>Banco<select name="banco_id" required data-zona="bancos"></select></label>
            <label>Nombre de la cuenta<input name="nombre" required placeholder="Ej. Cuenta operativa"></label>
            <div class="par">
              <label>Número de cuenta<input name="numero" required placeholder="000-000000-0"></label>
              <label>Tipo<select name="tipo">
                <option value="MONETARIA">Monetaria</option>
                <option value="AHORRO">Ahorro</option>
              </select></label>
            </div>
            <label>Saldo inicial<input name="saldo_inicial" type="number" step="0.01" value="0"></label>
            <div class="acciones">
              <button type="submit">Guardar cuenta</button>
              <button type="button" class="secundario oculto" data-accion="cancelar">Cancelar edición</button>
            </div>
          </form>
        </div>
        <div class="tarjeta">
          <h2>Cuentas registradas</h2>
          <div class="lista" data-zona="lista" style="margin-top:14px"></div>
        </div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="cuenta"]', (form) => this.guardar(form));
    this.alHacerClic('[data-accion="cancelar"]', () => this.limpiar());
    this.alHacerClic('[data-editar]', (boton) => this.editar(Number(boton.dataset.editar)));
    this.alHacerClic('[data-eliminar]', (boton) => this.eliminar(Number(boton.dataset.eliminar)));
    this.alHacerClic('[data-libro]', (boton) => {
      this.estado.elegirCuenta(boton.dataset.libro);
      this.ctx.abrirPestana('libro');
    });
  }

  async guardar(form) {
    const datos = datosFormulario(form);
    const carga = {
      cooperativa_id: Number(datos.cooperativa_id),
      banco_id: Number(datos.banco_id),
      nombre: datos.nombre,
      numero: datos.numero,
      tipo: datos.tipo || 'MONETARIA',
      saldo_inicial: Number(datos.saldo_inicial || 0),
    };
    try {
      if (datos.id) {
        carga.id = Number(datos.id);
        await api.actualizar('/api/cuentas', carga);
        exito('Cuenta actualizada');
      } else {
        await api.crear('/api/cuentas', carga);
        exito('Cuenta registrada');
      }
      this.limpiar();
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  editar(id) {
    const cuenta = this.estado.cuentas.find((c) => c.id === id);
    if (!cuenta) return;
    const form = this.$('[data-formulario="cuenta"]');
    form.id.value = cuenta.id;
    form.cooperativa_id.value = cuenta.cooperativa_id;
    form.banco_id.value = cuenta.banco_id;
    form.nombre.value = cuenta.nombre || '';
    form.numero.value = cuenta.numero;
    form.tipo.value = cuenta.tipo || 'MONETARIA';
    form.saldo_inicial.value = cuenta.saldo_inicial;
    this.$('[data-zona="titulo-formulario"]').textContent = `Editando cuenta ${cuenta.numero}`;
    this.$('[data-accion="cancelar"]').classList.remove('oculto');
    form.nombre.focus();
  }

  limpiar() {
    const form = this.$('[data-formulario="cuenta"]');
    form.reset();
    form.id.value = '';
    this.$('[data-zona="titulo-formulario"]').textContent = 'Nueva cuenta';
    this.$('[data-accion="cancelar"]').classList.add('oculto');
  }

  async eliminar(id) {
    const respuesta = await confirmar('Eliminar cuenta', 'Solo puede eliminarse una cuenta sin movimientos registrados.', {
      confirmar: 'Eliminar', peligro: true,
    });
    if (!respuesta) return;
    try {
      await api.eliminar(`/api/cuentas?id=${id}`);
      exito('Cuenta eliminada');
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async actualizar() {
    const form = this.$('[data-formulario="cuenta"]');
    const idCooperativa = form.cooperativa_id.value;
    const idBanco = form.banco_id.value;

    this.$('[data-zona="cooperativas"]').innerHTML = opciones(this.estado.cooperativas, {
      valor: (c) => c.id, texto: (c) => `${c.nombre} · NIT ${c.nit}`,
      seleccionado: idCooperativa || this.estado.cooperativaID, vacio: 'Selecciona una cooperativa',
    });
    this.$('[data-zona="bancos"]').innerHTML = opciones(this.estado.bancos, {
      valor: (b) => b.id, texto: (b) => b.nombre, seleccionado: idBanco, vacio: 'Selecciona un banco',
    });

    const lista = this.$('[data-zona="lista"]');
    if (!this.estado.cuentas.length) {
      lista.innerHTML = '<div class="vacio">Aún no hay cuentas. Necesitas al menos una cooperativa y un banco registrados.</div>';
      return;
    }
    lista.innerHTML = this.estado.cuentas.map((cuenta) => {
      const cooperativa = this.estado.cooperativas.find((c) => c.id === cuenta.cooperativa_id);
      const banco = this.estado.banco(cuenta.banco_id);
      return `
        <div class="elemento">
          <div>
            <strong>${escapar(cuenta.numero)} · ${escapar(cuenta.nombre || '')}</strong>
            <small>${escapar(cooperativa?.nombre || 'Sin cooperativa')} · ${escapar(banco?.nombre || 'Sin banco')} · ${escapar(cuenta.tipo || 'MONETARIA')}</small>
            <small class="cifra ${cuenta.saldo_actual < 0 ? 'negativo' : ''}">Saldo ${dinero(cuenta.saldo_actual)} · inicial ${dinero(cuenta.saldo_inicial)}</small>
          </div>
          <div class="controles">
            <button type="button" class="enlace" data-libro="${cuenta.id}">Ver libro</button>
            <button type="button" class="enlace" data-editar="${cuenta.id}">Editar</button>
            <button type="button" class="enlace" data-eliminar="${cuenta.id}">Eliminar</button>
          </div>
        </div>`;
    }).join('');
  }
}

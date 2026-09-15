import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar, datosFormulario } from '../core/dom.js';
import { dinero, fecha, hoy } from '../core/formato.js';
import { exito, problema, pendiente } from '../core/notificaciones.js';
import { pedirTexto, pedirFecha } from '../core/dialogo.js';

// Vista base de captura. Ingresos y Egresos son la misma pantalla con el tipo
// fijado, de modo que las reglas de validación viven en un solo lugar.
export class Movimientos extends Vista {
  static titulo = 'Movimientos';
  static glifo = '⇄';
  static tipoFijo = null;
  static necesitaCuenta = true;

  get tipo() {
    return this.constructor.tipoFijo;
  }

  plantilla() {
    const tipo = this.tipo;
    const selector = tipo ? `<input type="hidden" name="tipo" value="${tipo}">` : `
      <label>Tipo de movimiento
        <select name="tipo" data-campo="tipo" required>
          <option value="INGRESO">Ingreso / depósito</option>
          <option value="EGRESO">Egreso / cheque</option>
        </select>
      </label>`;

    return `
      ${this.encabezado(
        this.constructor.titulo,
        tipo === 'EGRESO'
          ? 'Cada cheque queda emitido hasta que el banco lo paga. El correlativo se verifica al guardar.'
          : tipo === 'INGRESO'
            ? 'Los depósitos suman al saldo desde la fecha de operación.'
            : 'Captura de ingresos y egresos sobre la cuenta seleccionada.',
      )}
      <div class="columnas">
        <div class="tarjeta">
          <h2>Registrar</h2>
          <form class="formulario" data-formulario="movimiento" style="margin-top:14px">
            ${selector}
            <div class="par">
              <label>Fecha de operación<input name="fecha_operacion" type="date" value="${hoy()}" required></label>
              <label data-zona="emision">Fecha de emisión<input name="fecha_emision_cheque" type="date"></label>
            </div>
            <label data-zona="documento">No. de cheque o documento<input name="numero_documento" autocomplete="off" placeholder="Ej. 000145231"></label>
            <label data-zona="remitente">Recibido de<input name="remitente" placeholder="Quién deposita"></label>
            <label data-zona="beneficiario">Beneficiario<input name="beneficiario" placeholder="A nombre de quién se emite"></label>
            <label>Concepto<input name="concepto" required placeholder="Ej. Pago de planillas"></label>
            <label>Monto<input name="monto" type="number" step="0.01" min="0.01" required placeholder="0.00"></label>
            <div class="acciones"><button type="submit">Registrar movimiento</button></div>
          </form>
        </div>
        <div class="tarjeta">
          <h2>Registrados en esta cuenta</h2>
          <div class="filtros" style="margin-top:14px">
            <label>Estado
              <select data-campo="estado">
                <option value="">Todos</option>
                <option value="ACTIVO">Activos</option>
                <option value="EMITIDO">Emitidos</option>
                <option value="COBRADO">Cobrados</option>
                <option value="ANULADO">Anulados</option>
              </select>
            </label>
            <button type="button" class="secundario" data-accion="recargar">Actualizar</button>
          </div>
          <div class="tabla-marco" data-zona="tabla"></div>
        </div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="movimiento"]', (form) => this.registrar(form));
    this.alHacerClic('[data-accion="recargar"]', () => this.actualizar());
    this.alHacerClic('[data-anular]', (boton) => this.anular(Number(boton.dataset.anular)));
    this.alHacerClic('[data-cobrar]', (boton) => this.cobrar(Number(boton.dataset.cobrar)));
    this.nodo.addEventListener('change', (evento) => {
      if (evento.target.matches('[data-campo="tipo"]')) this.ajustarCampos();
      if (evento.target.matches('[data-campo="estado"]')) this.actualizar();
    });
  }

  // Muestra solo los campos que corresponden al tipo de movimiento.
  ajustarCampos() {
    const form = this.$('[data-formulario="movimiento"]');
    const tipo = this.tipo || form.tipo.value;
    const esEgreso = tipo === 'EGRESO';
    this.$('[data-zona="remitente"]').classList.toggle('oculto', esEgreso);
    this.$('[data-zona="beneficiario"]').classList.toggle('oculto', !esEgreso);
    this.$('[data-zona="emision"]').classList.toggle('oculto', !esEgreso);
    this.$('[data-zona="documento"]').querySelector('input').required = esEgreso;
    this.$('[data-zona="documento"]').firstChild.textContent = esEgreso
      ? 'No. de cheque'
      : 'No. de boleta o documento';
  }

  async registrar(form) {
    const cuenta = this.estado.cuenta;
    if (!cuenta) {
      problema('Selecciona una cuenta en la barra superior antes de registrar.');
      return;
    }
    const datos = datosFormulario(form);
    const carga = {
      cuenta_id: cuenta.id,
      tipo: this.tipo || datos.tipo,
      fecha_operacion: datos.fecha_operacion || '',
      fecha_emision_cheque: datos.fecha_emision_cheque || '',
      numero_documento: datos.numero_documento || '',
      remitente: datos.remitente || '',
      beneficiario: datos.beneficiario || '',
      concepto: datos.concepto,
      monto: Number(datos.monto),
    };

    try {
      const respuesta = await api.crear('/api/movimientos', carga);
      exito(`Movimiento registrado por ${dinero(respuesta.movimiento.monto)}`);
      (respuesta.avisos || []).forEach((aviso) => pendiente(aviso.mensaje));
      form.reset();
      form.fecha_operacion.value = hoy();
      this.ajustarCampos();
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async anular(id) {
    const motivo = await pedirTexto(
      'Anular movimiento',
      'Motivo de la anulación (queda en la bitácora)',
      { confirmar: 'Anular', peligro: true },
    );
    if (!motivo) return;
    try {
      await api.eliminar(`/api/movimientos?${consulta({ id, motivo })}`);
      exito('Movimiento anulado');
      await this.ctx.refrescarCatalogos();
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async cobrar(id) {
    const fechaCobro = await pedirFecha('Registrar cobro del cheque', 'Fecha en que el banco lo pagó', hoy());
    if (!fechaCobro) return;
    try {
      await api.crear('/api/movimientos/cobrar', { id, fecha_cobro: fechaCobro });
      exito('Cheque marcado como cobrado');
      await this.actualizar();
    } catch (error) {
      problema(error.message);
    }
  }

  async actualizar() {
    this.ajustarCampos();
    const tabla = this.$('[data-zona="tabla"]');
    const cuenta = this.estado.cuenta;
    if (!cuenta) {
      tabla.innerHTML = '<div class="vacio">Elige una cuenta en la barra de contexto para ver sus movimientos.</div>';
      return;
    }

    const movimientos = await api.obtener(`/api/movimientos?${consulta({
      cuenta_id: cuenta.id,
      tipo: this.tipo || '',
      estado: this.$('[data-campo="estado"]').value,
      limite: 200,
    })}`);

    if (!movimientos.length) {
      tabla.innerHTML = '<div class="vacio">No hay movimientos que coincidan con el filtro.</div>';
      return;
    }

    tabla.innerHTML = `
      <table>
        <thead><tr>
          <th>Fecha</th><th>Documento</th><th>Concepto</th><th>Persona</th>
          <th class="cifra">Monto</th><th>Estado</th><th></th>
        </tr></thead>
        <tbody>${movimientos.map((m) => `
          <tr class="${m.estado === 'ANULADO' ? 'anulada' : ''}">
            <td>${fecha(m.fecha_operacion)}</td>
            <td>${escapar(m.numero_documento || '—')}</td>
            <td>${escapar(m.concepto)}${m.motivo_anulacion ? `<br><small class="tenue">Anulado: ${escapar(m.motivo_anulacion)}</small>` : ''}</td>
            <td>${escapar(m.beneficiario || m.remitente || '')}</td>
            <td class="cifra ${m.tipo === 'INGRESO' ? 'deposito' : 'cheque'}">${m.tipo === 'INGRESO' ? '' : '−'}${dinero(m.monto)}</td>
            <td><span class="etiqueta ${m.estado.toLowerCase()}">${escapar(m.estado)}</span>${m.fecha_cobro ? `<br><small class="tenue">${fecha(m.fecha_cobro)}</small>` : ''}</td>
            <td>
              ${m.tipo === 'EGRESO' && m.estado === 'EMITIDO' ? `<button type="button" class="enlace" data-cobrar="${m.id}">Marcar cobrado</button>` : ''}
              ${m.estado !== 'ANULADO' && m.estado !== 'COBRADO' ? `<button type="button" class="enlace" data-anular="${m.id}">Anular</button>` : ''}
            </td>
          </tr>`).join('')}</tbody>
      </table>`;
  }
}

export class Ingresos extends Movimientos {
  static titulo = 'Ingresos';
  static glifo = '↓';
  static tipoFijo = 'INGRESO';
}

export class Egresos extends Movimientos {
  static titulo = 'Egresos';
  static glifo = '↑';
  static tipoFijo = 'EGRESO';
}

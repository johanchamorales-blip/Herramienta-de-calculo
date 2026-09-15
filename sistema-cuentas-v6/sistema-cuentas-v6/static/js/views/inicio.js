import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { dinero, fecha, periodo, MESES } from '../core/formato.js';
import { escapar } from '../core/dom.js';

// Panel de apertura: el estado de las cuentas del mes y lo que necesita atención.
export class Inicio extends Vista {
  static titulo = 'Panel';
  static glifo = '◈';

  plantilla() {
    const ahora = new Date();
    return `
      ${this.encabezado('Panel del mes', 'Saldos, movimientos del periodo y pendientes de revisión.')}
      <div class="filtros">
        <label>Mes
          <select data-campo="mes">
            ${MESES.map((nombre, i) => `<option value="${i + 1}"${i === ahora.getMonth() ? ' selected' : ''}>${nombre}</option>`).join('')}
          </select>
        </label>
        <label>Año
          <input type="number" data-campo="anio" min="2000" max="2100" value="${ahora.getFullYear()}">
        </label>
        <button type="button" data-accion="consultar">Ver periodo</button>
      </div>
      <div data-zona="alertas"></div>
      <div class="tarjeta">
        <h2>Resumen</h2>
        <div class="metricas" data-zona="metricas"></div>
      </div>
      <div class="tarjeta">
        <h2>Cuentas</h2>
        <div class="tabla-marco" data-zona="cuentas"></div>
      </div>
      <div class="tarjeta">
        <h2>Últimos movimientos</h2>
        <div class="tabla-marco" data-zona="movimientos"></div>
      </div>`;
  }

  conectar() {
    this.alHacerClic('[data-accion="consultar"]', () => this.actualizar());
  }

  async actualizar() {
    const mes = this.$('[data-campo="mes"]').value;
    const anio = this.$('[data-campo="anio"]').value;
    const datos = await api.obtener(`/api/resumen?${consulta({ mes, anio })}`);
    const t = datos.totales;

    this.$('[data-zona="alertas"]').innerHTML = datos.alertas.map((alerta) => `
      <div class="aviso pendiente"><span aria-hidden="true">▲</span><span>${escapar(alerta.mensaje)}</span></div>`).join('');

    this.$('[data-zona="metricas"]').innerHTML = `
      <div class="metrica"><span>Saldo en cuentas</span><strong class="cifra">${dinero(t.saldo)}</strong><small>${t.cuentas} cuenta(s) en ${t.cooperativas} cooperativa(s)</small></div>
      <div class="metrica"><span>Depósitos de ${periodo(datos.periodo.mes, datos.periodo.anio)}</span><strong class="cifra deposito">${dinero(t.depositos_mes)}</strong></div>
      <div class="metrica"><span>Cheques del periodo</span><strong class="cifra cheque">${dinero(t.cheques_mes)}</strong></div>
      <div class="metrica"><span>Cheques en circulación</span><strong class="cifra">${dinero(t.monto_en_circulacion)}</strong><small>Pendientes de cobro en el banco</small></div>`;

    this.$('[data-zona="cuentas"]').innerHTML = datos.cuentas.length ? `
      <table>
        <thead><tr>
          <th>Cuenta</th><th>Cooperativa</th><th>Banco</th>
          <th class="cifra">Depósitos</th><th class="cifra">Cheques</th>
          <th class="cifra">Saldo</th><th>Conciliada hasta</th>
        </tr></thead>
        <tbody>${datos.cuentas.map((c) => `
          <tr>
            <td><strong>${escapar(c.numero)}</strong><br><small class="tenue">${escapar(c.nombre)}</small></td>
            <td>${escapar(c.cooperativa)}<br><small class="tenue">${escapar(c.nit)}</small></td>
            <td>${escapar(c.banco)}</td>
            <td class="cifra deposito">${dinero(c.depositos_mes)}</td>
            <td class="cifra cheque">${dinero(c.cheques_mes)}</td>
            <td class="cifra ${c.saldo < 0 ? 'negativo' : ''}">${dinero(c.saldo)}</td>
            <td>${c.conciliada_hasta ? escapar(c.conciliada_hasta) : '<span class="etiqueta emitido">Sin conciliar</span>'}</td>
          </tr>`).join('')}</tbody>
      </table>` : '<div class="vacio">Registra una cooperativa y su cuenta bancaria para empezar.</div>';

    const movimientos = datos.ultimos_movimientos || [];
    this.$('[data-zona="movimientos"]').innerHTML = movimientos.length ? `
      <table>
        <thead><tr><th>Fecha</th><th>Documento</th><th>Concepto</th><th>A favor de / recibido de</th><th class="cifra">Monto</th><th>Estado</th></tr></thead>
        <tbody>${movimientos.map((m) => `
          <tr>
            <td>${fecha(m.fecha_operacion)}</td>
            <td>${escapar(m.numero_documento || '—')}</td>
            <td>${escapar(m.concepto)}</td>
            <td>${escapar(m.beneficiario || m.remitente || '')}</td>
            <td class="cifra ${m.tipo === 'INGRESO' ? 'deposito' : 'cheque'}">${m.tipo === 'INGRESO' ? '' : '−'}${dinero(m.monto)}</td>
            <td><span class="etiqueta ${m.estado.toLowerCase()}">${escapar(m.estado)}</span></td>
          </tr>`).join('')}</tbody>
      </table>` : '<div class="vacio">Todavía no hay movimientos registrados.</div>';
  }
}

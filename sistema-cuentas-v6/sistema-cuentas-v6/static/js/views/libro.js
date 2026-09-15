import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar, opcionesMes } from '../core/dom.js';
import { dinero, fecha, periodo } from '../core/formato.js';
import { problema } from '../core/notificaciones.js';

// Libro de bancos: la tabla con saldo corrido y, al imprimir, el documento
// formal con encabezado de la cooperativa y las cuatro firmas.
export class LibroBancos extends Vista {
  static titulo = 'Libro de bancos';
  static glifo = '≡';
  static necesitaCuenta = true;

  constructor(contexto) {
    super(contexto);
    this.pagina = 1;
  }

  plantilla() {
    const ahora = new Date();
    return `
      ${this.encabezado('Libro de bancos', 'Saldo corrido por cuenta y periodo. Imprime con el formato oficial de la cooperativa.', `
        <button type="button" class="secundario" data-accion="csv">Exportar CSV</button>
        <button type="button" data-accion="imprimir">Imprimir libro</button>`)}

      <div class="filtros">
        <label>Mes<select data-campo="mes">${opcionesMes(ahora.getMonth() + 1, true)}</select></label>
        <label>Año<input type="number" data-campo="anio" min="2000" max="2100" value="${ahora.getFullYear()}"></label>
        <button type="button" data-accion="consultar">Consultar</button>
      </div>

      <div class="tarjeta">
        <div class="documento" data-zona="documento"></div>
        <div class="metricas" data-zona="metricas"></div>
        <div class="tabla-marco" style="margin-top:14px" data-zona="tabla"></div>
        <div class="paginacion">
          <span data-zona="paginas">Página 0 de 0</span>
          <div class="acciones">
            <button type="button" class="secundario" data-accion="anterior">Anterior</button>
            <button type="button" class="secundario" data-accion="siguiente">Siguiente</button>
          </div>
        </div>
        <div class="firmas">
          <div>Elaboró</div><div>Tesorero</div><div>Vo. Bo.</div><div>Presidente Comisión de Vigilancia</div>
        </div>
      </div>`;
  }

  conectar() {
    this.alHacerClic('[data-accion="consultar"]', () => { this.pagina = 1; this.actualizar(); });
    this.alHacerClic('[data-accion="anterior"]', () => { if (this.pagina > 1) { this.pagina -= 1; this.actualizar(); } });
    this.alHacerClic('[data-accion="siguiente"]', () => { this.pagina += 1; this.actualizar(); });
    this.alHacerClic('[data-accion="imprimir"]', () => window.print());
    this.alHacerClic('[data-accion="csv"]', () => this.exportar());
  }

  filtros() {
    const mes = this.$('[data-campo="mes"]').value;
    const anio = this.$('[data-campo="anio"]').value;
    return { mes: mes || '', anio: mes ? anio : '' };
  }

  exportar() {
    const cuenta = this.estado.cuenta;
    if (!cuenta) {
      problema('Selecciona una cuenta antes de exportar.');
      return;
    }
    window.location.href = `/api/libro-bancos.csv?${consulta({ cuenta_id: cuenta.id, ...this.filtros() })}`;
  }

  async actualizar() {
    const cuenta = this.estado.cuenta;
    const tabla = this.$('[data-zona="tabla"]');
    if (!cuenta) {
      tabla.innerHTML = '<div class="vacio">Elige una cuenta en la barra de contexto para ver su libro.</div>';
      this.$('[data-zona="metricas"]').innerHTML = '';
      return;
    }

    const reporte = await api.obtener(`/api/libro-bancos?${consulta({
      cuenta_id: cuenta.id, ...this.filtros(), pagina: this.pagina, tamano: 50,
    })}`);
    const meta = reporte.meta;

    this.$('[data-zona="documento"]').innerHTML = `
      <h1>${escapar(meta.cooperativa || 'Cooperativa')}</h1>
      <p>NIT ${escapar(meta.nit)}${meta.direccion ? ` · ${escapar(meta.direccion)}` : ''}</p>
      <p>Libro de bancos · ${escapar(meta.nombre_cuenta || '')} No. ${escapar(meta.numero_cuenta)} · ${escapar(meta.banco)}</p>
      <p>Periodo: ${escapar(periodo(meta.mes, meta.anio))}</p>`;

    this.$('[data-zona="metricas"]').innerHTML = `
      <div class="metrica"><span>Saldo inicial</span><strong class="cifra">${dinero(meta.saldo_inicial_periodo)}</strong></div>
      <div class="metrica"><span>Depósitos</span><strong class="cifra deposito">${dinero(meta.total_depositos)}</strong></div>
      <div class="metrica"><span>Cheques</span><strong class="cifra cheque">${dinero(meta.total_cheques)}</strong></div>
      <div class="metrica"><span>Saldo final</span><strong class="cifra ${meta.saldo_final_periodo < 0 ? 'negativo' : ''}">${dinero(meta.saldo_final_periodo)}</strong></div>
      <div class="metrica"><span>En circulación</span><strong class="cifra">${dinero(meta.cheques_en_circulacion)}</strong><small>Cheques emitidos sin cobrar</small></div>`;

    tabla.innerHTML = reporte.filas.length ? `
      <table>
        <thead><tr>
          <th>No.</th><th>Fecha</th><th>No. cheque o documento</th><th>Beneficiario</th>
          <th>Concepto</th><th class="cifra">Depósitos</th><th class="cifra">Cheques</th><th class="cifra">Saldo</th>
        </tr></thead>
        <tbody>${reporte.filas.map((f) => `
          <tr>
            <td>${f.numero}</td>
            <td>${fecha(f.fecha_operacion)}</td>
            <td>${escapar(f.numero_documento || '')}</td>
            <td>${escapar(f.beneficiario || f.remitente || '')}</td>
            <td>${escapar(f.concepto)}</td>
            <td class="cifra deposito">${f.deposito ? dinero(f.deposito) : ''}</td>
            <td class="cifra cheque">${f.cheque ? dinero(f.cheque) : ''}</td>
            <td class="cifra ${f.saldo_actual < 0 ? 'negativo' : ''}">${dinero(f.saldo_actual)}</td>
          </tr>`).join('')}</tbody>
        <tfoot><tr>
          <td colspan="5">Totales del periodo</td>
          <td class="cifra">${dinero(meta.total_depositos)}</td>
          <td class="cifra">${dinero(meta.total_cheques)}</td>
          <td class="cifra">${dinero(meta.saldo_final_periodo)}</td>
        </tr></tfoot>
      </table>` : '<div class="vacio">No hay movimientos en el periodo seleccionado.</div>';

    this.pagina = meta.total_paginas ? meta.pagina : 1;
    this.$('[data-zona="paginas"]').textContent =
      `Página ${meta.total_paginas ? meta.pagina : 0} de ${meta.total_paginas} · ${meta.total_registros} registro(s)`;
    this.$('[data-accion="anterior"]').disabled = meta.pagina <= 1;
    this.$('[data-accion="siguiente"]').disabled = !meta.total_paginas || meta.pagina >= meta.total_paginas;
  }
}

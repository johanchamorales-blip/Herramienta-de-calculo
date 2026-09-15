import { Vista } from '../core/vista.js';
import { api } from '../core/api.js';
import { escapar } from '../core/dom.js';
import { dinero, fecha } from '../core/formato.js';
import { problema } from '../core/notificaciones.js';

// Búsqueda tolerante a errores de escritura sobre cooperativas, NIT, cuentas
// y movimientos.
export class Busqueda extends Vista {
  static titulo = 'Búsqueda';
  static glifo = '⌕';

  plantilla() {
    return `
      ${this.encabezado('Búsqueda', 'Escribe un nombre, un NIT, un número de cuenta o un beneficiario. Tolera errores de escritura.')}
      <div class="tarjeta">
        <form class="filtros" data-formulario="busqueda" style="margin-bottom:0">
          <label style="flex:1; min-width:260px">Qué buscas
            <input name="q" required placeholder="Ej. Chicoj, 1234567-8, 000145231, Gasolinera" autocomplete="off">
          </label>
          <button type="submit">Buscar</button>
        </form>
      </div>
      <div data-zona="resultados"></div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="busqueda"]', (form) => this.buscar(form.q.value.trim()));
    this.alHacerClic('[data-cuenta]', (boton) => {
      this.estado.elegirCuenta(boton.dataset.cuenta);
      this.ctx.abrirPestana('libro');
    });
  }

  async buscar(texto) {
    if (!texto) return;
    try {
      const resultado = await api.obtener(`/api/busqueda?q=${encodeURIComponent(texto)}`);
      this.pintar(resultado);
    } catch (error) {
      problema(error.message);
    }
  }

  pintar(resultado) {
    const zona = this.$('[data-zona="resultados"]');
    const bloques = [];

    if (resultado.cooperativas?.length) {
      bloques.push(`
        <div class="tarjeta">
          <h2>Cooperativas</h2>
          <div class="lista" style="margin-top:12px">
            ${resultado.cooperativas.map((r) => `
              <div class="elemento">
                <div>
                  <strong>${escapar(r.cooperativa.nombre)}</strong>
                  <small>NIT ${escapar(r.cooperativa.nit)}${r.cooperativa.direccion ? ` · ${escapar(r.cooperativa.direccion)}` : ''}</small>
                  ${(r.bancos || []).map((b) => `
                    <small>${escapar(b.banco.nombre)}: ${b.cuentas.map((c) => `cuenta ${escapar(c.cuenta.numero)} (${dinero(c.cuenta.saldo_actual)}, ${c.libros_movimientos} movs.)`).join(' · ')}</small>`).join('')}
                </div>
                <span class="etiqueta">${r.puntaje}% de coincidencia</span>
              </div>`).join('')}
          </div>
        </div>`);
    }

    if (resultado.cuentas?.length) {
      bloques.push(`
        <div class="tarjeta">
          <h2>Cuentas</h2>
          <div class="lista" style="margin-top:12px">
            ${resultado.cuentas.map((r) => `
              <div class="elemento">
                <div>
                  <strong>${escapar(r.cuenta.numero)}</strong>
                  <small>${escapar(r.cooperativa.nombre)} · ${escapar(r.banco.nombre)} · saldo ${dinero(r.cuenta.saldo_actual)}</small>
                  <small>${r.libros} movimiento(s) · ${r.conciliaciones} conciliación(es)</small>
                </div>
                <button type="button" class="enlace" data-cuenta="${r.cuenta.id}">Abrir libro</button>
              </div>`).join('')}
          </div>
        </div>`);
    }

    if (resultado.movimientos?.length) {
      bloques.push(`
        <div class="tarjeta">
          <h2>Movimientos</h2>
          <div class="tabla-marco" style="margin-top:12px">
            <table>
              <thead><tr><th>Fecha</th><th>Documento</th><th>Concepto</th><th>Cuenta</th><th class="cifra">Monto</th><th>Estado</th></tr></thead>
              <tbody>${resultado.movimientos.slice(0, 40).map((r) => `
                <tr>
                  <td>${fecha(r.movimiento.fecha_operacion)}</td>
                  <td>${escapar(r.movimiento.numero_documento || '—')}</td>
                  <td>${escapar(r.movimiento.concepto)}<br><small class="tenue">${escapar(r.movimiento.beneficiario || r.movimiento.remitente || '')}</small></td>
                  <td>${escapar(r.cuenta.numero)}<br><small class="tenue">${escapar(r.cooperativa.nombre)}</small></td>
                  <td class="cifra ${r.movimiento.tipo === 'INGRESO' ? 'deposito' : 'cheque'}">${dinero(r.movimiento.monto)}</td>
                  <td><span class="etiqueta ${r.movimiento.estado.toLowerCase()}">${escapar(r.movimiento.estado)}</span></td>
                </tr>`).join('')}</tbody>
            </table>
          </div>
        </div>`);
    }

    zona.innerHTML = bloques.join('') || '<div class="vacio">Sin coincidencias. Prueba con menos palabras o con el número de cuenta.</div>';
  }

  async actualizar() {}
}

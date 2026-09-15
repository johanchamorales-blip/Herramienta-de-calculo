import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar } from '../core/dom.js';
import { fechaHora, nombreRol } from '../core/formato.js';

// Bitácora: quién hizo qué y cuándo. Es el respaldo ante una auditoría.
export class Auditoria extends Vista {
  static titulo = 'Bitácora';
  static glifo = '◷';

  plantilla() {
    return `
      ${this.encabezado('Bitácora de auditoría', 'Registro de altas, anulaciones, cobros y conciliaciones.', `
        <button type="button" class="secundario" data-accion="recargar">Actualizar</button>`)}
      <div class="tarjeta">
        <div class="filtros">
          <label style="flex:1; min-width:220px">Buscar<input data-campo="q" placeholder="Usuario, acción o documento" autocomplete="off"></label>
          <label>Mostrar
            <select data-campo="limite">
              <option value="100">Últimos 100</option>
              <option value="200" selected>Últimos 200</option>
              <option value="500">Últimos 500</option>
            </select>
          </label>
        </div>
        <div class="tabla-marco" data-zona="tabla"></div>
      </div>`;
  }

  conectar() {
    this.alHacerClic('[data-accion="recargar"]', () => this.actualizar());
    let temporizador;
    this.nodo.addEventListener('input', (evento) => {
      if (!evento.target.matches('[data-campo="q"]')) return;
      clearTimeout(temporizador);
      temporizador = setTimeout(() => this.actualizar(), 320);
    });
    this.nodo.addEventListener('change', (evento) => {
      if (evento.target.matches('[data-campo="limite"]')) this.actualizar();
    });
  }

  async actualizar() {
    const eventos = await api.obtener(`/api/auditoria?${consulta({
      q: this.$('[data-campo="q"]').value.trim(),
      limite: this.$('[data-campo="limite"]').value,
    })}`);

    this.$('[data-zona="tabla"]').innerHTML = eventos.length ? `
      <table>
        <thead><tr><th>Fecha y hora</th><th>Usuario</th><th>Acción</th><th>Documento</th><th>Detalle</th></tr></thead>
        <tbody>${eventos.map((e) => `
          <tr>
            <td>${fechaHora(e.fecha_hora)}</td>
            <td>${escapar(e.usuario)}${e.rol ? `<br><small class="tenue">${escapar(nombreRol(e.rol))}</small>` : ''}</td>
            <td>${escapar(e.accion)}</td>
            <td>${escapar(e.documento_afectado || '')}</td>
            <td>${escapar(e.detalle || '')}</td>
          </tr>`).join('')}</tbody>
      </table>` : '<div class="vacio">No hay eventos registrados todavía.</div>';
  }
}

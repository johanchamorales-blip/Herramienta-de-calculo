import { escapar } from './dom.js';

// Diálogo modal que devuelve una promesa. Sustituye a confirm() y prompt()
// para poder pedir, por ejemplo, el motivo de una anulación.
function abrir({ titulo, cuerpo, confirmar = 'Confirmar', cancelar = 'Cancelar', peligro = false }) {
  return new Promise((resolver) => {
    const velo = document.createElement('div');
    velo.className = 'velo';
    velo.innerHTML = `
      <div class="dialogo" role="dialog" aria-modal="true" aria-label="${escapar(titulo)}">
        <h2>${escapar(titulo)}</h2>
        <div class="cuerpo-dialogo">${cuerpo}</div>
        <div class="acciones">
          <button type="button" class="secundario" data-accion="cancelar">${escapar(cancelar)}</button>
          <button type="button" class="${peligro ? 'peligro' : ''}" data-accion="confirmar">${escapar(confirmar)}</button>
        </div>
      </div>`;

    const cerrar = (resultado) => {
      document.removeEventListener('keydown', alPresionar);
      velo.remove();
      resolver(resultado);
    };
    const alPresionar = (evento) => {
      if (evento.key === 'Escape') cerrar(null);
    };

    velo.addEventListener('click', (evento) => {
      if (evento.target === velo) cerrar(null);
      const accion = evento.target.dataset?.accion;
      if (accion === 'cancelar') cerrar(null);
      if (accion === 'confirmar') {
        const campo = velo.querySelector('[data-campo]');
        cerrar(campo ? campo.value.trim() : true);
      }
    });

    document.addEventListener('keydown', alPresionar);
    document.body.appendChild(velo);
    const foco = velo.querySelector('[data-campo]') || velo.querySelector('[data-accion="confirmar"]');
    foco?.focus();
  });
}

export function confirmar(titulo, mensaje, opciones = {}) {
  return abrir({ titulo, cuerpo: `<p>${escapar(mensaje)}</p>`, ...opciones });
}

export function pedirTexto(titulo, etiqueta, opciones = {}) {
  return abrir({
    titulo,
    cuerpo: `<label>${escapar(etiqueta)}<input data-campo type="text" autocomplete="off"></label>`,
    ...opciones,
  });
}

export function pedirFecha(titulo, etiqueta, valorInicial) {
  return abrir({
    titulo,
    cuerpo: `<label>${escapar(etiqueta)}<input data-campo type="date" value="${escapar(valorInicial)}"></label>`,
    confirmar: 'Registrar',
  });
}

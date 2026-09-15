// Clase base de todas las pestañas. Cada módulo hereda de aquí, de modo que la
// pantalla principal no crece: se agrega un archivo por pestaña.
export class Vista {
  static titulo = 'Pestaña';
  static glifo = '•';
  static descripcion = '';
  static necesitaCuenta = false;

  constructor(contexto) {
    this.ctx = contexto;           // { estado, abrirPestana, refrescar }
    this.estado = contexto.estado;
    this.nodo = document.createElement('section');
    this.nodo.className = 'vista';
  }

  // Se llama una sola vez, al abrir la pestaña.
  async montar() {
    this.nodo.innerHTML = this.plantilla();
    this.conectar();
    await this.actualizar();
  }

  // HTML inicial de la pestaña.
  plantilla() {
    return '';
  }

  // Enlaza eventos una sola vez, después de pintar la plantilla.
  conectar() {}

  // Recarga los datos. Se llama al montar, al cambiar de cuenta y al refrescar.
  async actualizar() {}

  // Libera lo que haga falta cuando se cierra la pestaña.
  destruir() {}

  encabezado(titulo, descripcion, acciones = '') {
    return `
      <div class="encabezado-vista">
        <div>
          <h1>${titulo}</h1>
          ${descripcion ? `<p>${descripcion}</p>` : ''}
        </div>
        <div class="acciones">${acciones}</div>
      </div>`;
  }

  $(selector) {
    return this.nodo.querySelector(selector);
  }

  $$(selector) {
    return Array.from(this.nodo.querySelectorAll(selector));
  }

  // Delegación de eventos: un solo oyente por pestaña.
  alHacerClic(selector, manejador) {
    this.nodo.addEventListener('click', (evento) => {
      const objetivo = evento.target.closest(selector);
      if (objetivo && this.nodo.contains(objetivo)) manejador(objetivo, evento);
    });
  }

  alEnviar(selector, manejador) {
    this.nodo.addEventListener('submit', async (evento) => {
      if (!evento.target.matches(selector)) return;
      evento.preventDefault();
      const boton = evento.target.querySelector('button[type="submit"], button:not([type])');
      if (boton) boton.disabled = true;
      try {
        await manejador(evento.target);
      } finally {
        if (boton) boton.disabled = false;
      }
    });
  }
}

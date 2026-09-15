// Utilidades mínimas de DOM. Todo lo que venga de datos pasa por escapar().
export function escapar(valor) {
  if (valor === null || valor === undefined) return '';
  return String(valor)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

export function crear(etiqueta, clases = '', html = '') {
  const nodo = document.createElement(etiqueta);
  if (clases) nodo.className = clases;
  if (html) nodo.innerHTML = html;
  return nodo;
}

export function opciones(items, { valor, texto, seleccionado, vacio }) {
  const partes = [];
  if (vacio) partes.push(`<option value="">${escapar(vacio)}</option>`);
  items.forEach((item) => {
    const v = valor(item);
    const marca = String(v) === String(seleccionado) ? ' selected' : '';
    partes.push(`<option value="${escapar(v)}"${marca}>${escapar(texto(item))}</option>`);
  });
  return partes.join('');
}

export function opcionesMes(seleccionado, incluirTodos = false) {
  const meses = ['Enero', 'Febrero', 'Marzo', 'Abril', 'Mayo', 'Junio', 'Julio', 'Agosto', 'Septiembre', 'Octubre', 'Noviembre', 'Diciembre'];
  const partes = incluirTodos ? ['<option value="">Todos los meses</option>'] : [];
  meses.forEach((nombre, i) => {
    const valor = i + 1;
    partes.push(`<option value="${valor}"${String(valor) === String(seleccionado) ? ' selected' : ''}>${nombre}</option>`);
  });
  return partes.join('');
}

// Lee un formulario como objeto plano, sin valores vacíos.
export function datosFormulario(formulario) {
  const datos = {};
  new FormData(formulario).forEach((valor, clave) => {
    const texto = typeof valor === 'string' ? valor.trim() : valor;
    if (texto !== '') datos[clave] = texto;
  });
  return datos;
}

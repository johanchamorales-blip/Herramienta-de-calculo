// Cliente HTTP del sistema. Centraliza el manejo de errores para que las
// vistas solo se ocupen de qué pedir, no de cómo.
const oyentesSinSesion = new Set();

export function alPerderSesion(fn) {
  oyentesSinSesion.add(fn);
}

export class ErrorAPI extends Error {
  constructor(mensaje, estado) {
    super(mensaje);
    this.estado = estado;
  }
}

async function peticion(url, opciones = {}) {
  const respuesta = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    ...opciones,
  });

  const tipo = respuesta.headers.get('content-type') || '';
  const cuerpo = tipo.includes('application/json') ? await respuesta.json() : await respuesta.text();

  if (respuesta.status === 401) {
    oyentesSinSesion.forEach((fn) => fn());
    throw new ErrorAPI('Tu sesión expiró. Vuelve a iniciar sesión.', 401);
  }
  if (!respuesta.ok) {
    const mensaje = (cuerpo && cuerpo.error) || 'No se pudo completar la operación';
    throw new ErrorAPI(mensaje, respuesta.status);
  }
  return cuerpo;
}

export const api = {
  obtener: (url) => peticion(url),
  crear: (url, datos) => peticion(url, { method: 'POST', body: JSON.stringify(datos) }),
  actualizar: (url, datos) => peticion(url, { method: 'PUT', body: JSON.stringify(datos) }),
  parchear: (url, datos) => peticion(url, { method: 'PATCH', body: JSON.stringify(datos) }),
  eliminar: (url) => peticion(url, { method: 'DELETE' }),
};

// Construye una query string omitiendo valores vacíos.
export function consulta(parametros) {
  const params = new URLSearchParams();
  Object.entries(parametros).forEach(([clave, valor]) => {
    if (valor !== undefined && valor !== null && valor !== '') params.set(clave, valor);
  });
  return params.toString();
}

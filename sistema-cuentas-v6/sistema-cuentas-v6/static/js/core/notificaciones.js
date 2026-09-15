// Mensajes breves en la esquina inferior. Duran lo suficiente para leerse
// y no bloquean la pantalla como hacía alert().
const contenedor = document.createElement('div');
contenedor.className = 'notificaciones';
contenedor.setAttribute('role', 'status');
contenedor.setAttribute('aria-live', 'polite');
document.body.appendChild(contenedor);

export function notificar(mensaje, tono = 'neutro', duracion = 4200) {
  const nodo = document.createElement('div');
  nodo.className = `notificacion ${tono}`;
  nodo.textContent = mensaje;
  contenedor.appendChild(nodo);
  setTimeout(() => nodo.remove(), duracion);
}

export const exito = (mensaje) => notificar(mensaje, 'logro');
export const problema = (mensaje) => notificar(mensaje, 'error', 6000);
export const pendiente = (mensaje) => notificar(mensaje, 'pendiente', 7000);
